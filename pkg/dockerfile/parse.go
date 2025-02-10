package dockerfile

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/loft-sh/log/scanner"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
)

var argumentExpression = regexp.MustCompile(`(?m)\$\{?([a-zA-Z0-9_]+)(:(-|\+)([^\}]+))?\}?`)

var dockerfileSyntax = regexp.MustCompile(`(?m)^[\s\t]*#[\s\t]*syntax=.*$`)

func (d *Dockerfile) FindUserStatement(buildArgs map[string]string, baseImageEnv map[string]string, target string) string {
	stage, ok := d.StagesByTarget[target]
	if !ok {
		stage = d.Stages[len(d.Stages)-1]
	}

	seenStages := map[string]bool{}
	for {
		if seenStages[stageID(&stage.BaseStage)] {
			return ""
		}
		seenStages[stageID(&stage.BaseStage)] = true

		if len(stage.Users) > 0 {
			lastUser := stage.Users[len(stage.Users)-1]
			return d.replaceVariables(lastUser.Key, buildArgs, baseImageEnv, &stage.BaseStage, lastUser.Line)
		}

		// is preamble?
		if stage.Image == "" {
			return ""
		}

		image := d.replaceVariables(stage.Image, buildArgs, baseImageEnv, &d.Preamble.BaseStage, d.Stages[0].Instructions[0].StartLine)
		stage, ok = d.StagesByTarget[image]
		if !ok {
			return ""
		}
	}
}

func stageID(stage *BaseStage) string {
	return stage.Image + "-" + stage.Target
}

func (d *Dockerfile) FindBaseImage(buildArgs map[string]string, target string) string {
	stage, ok := d.StagesByTarget[target]
	if !ok {
		stage = d.Stages[len(d.Stages)-1]
	}

	visited := map[*Stage]bool{}
	for stage != nil {
		if visited[stage] {
			return ""
		}
		visited[stage] = true

		nextTarget := d.replaceVariables(stage.Image, buildArgs, nil, &d.Preamble.BaseStage, d.Stages[0].Instructions[0].StartLine)
		nextStage := d.StagesByTarget[nextTarget]
		if nextStage == nil {
			return nextTarget
		}

		stage = nextStage
	}

	return ""
}

// BuildContextFiles traverses a build stage and returns a list of any file path that would affect the build context
func (d *Dockerfile) BuildContextFiles() (files []string) {
	// Iterate over all build stages
	for _, stage := range d.Stages {
		// Add the values of any ADD or COPY instructions
		for _, in := range stage.Instructions {
			if strings.HasPrefix(in.Value, "ADD") || strings.HasPrefix(in.Value, "COPY") {
				// Take all parts except the first (ADD/COPY) and the last (destination on remote), e.g. "COPY src files /app", we want src and files
				parts := strings.Split(in.Original, " ")
				if len(parts) > 2 {
					files = append(files, parts[1:len(parts)-1]...)
				}
			}
		}
	}
	return files
}

func (d *Dockerfile) replaceVariables(val string, buildArgs map[string]string, baseImageEnv map[string]string, stage *BaseStage, untilLine int) string {
	newVal := argumentExpression.ReplaceAllFunc([]byte(val), func(match []byte) []byte {
		subMatches := argumentExpression.FindStringSubmatch(string(match))
		variable := subMatches[1]
		value := d.findValue(buildArgs, baseImageEnv, variable, stage, untilLine)

		// is expression?
		if subMatches[2] != "" {
			value = getExpressionValue(subMatches[3], value != "", subMatches[4], value)
		}

		return []byte(value)
	})

	return string(newVal)
}

func getExpressionValue(option string, isSet bool, defaultValue string, value string) string {
	output := ""
	if option == "-" {
		if isSet {
			output = value
		} else {
			output = defaultValue
		}
	} else if option == "+" {
		if isSet {
			output = defaultValue
		} else {
			output = value
		}
	}

	return strings.Trim(output, "\"'")
}

func (d *Dockerfile) findValue(buildArgs map[string]string, baseImageEnv map[string]string, variable string, stage *BaseStage, untilLine int) string {
	considerArg := true
	if buildArgs == nil {
		buildArgs = map[string]string{}
	}

	seenStages := map[string]bool{}
	for {
		if seenStages[stageID(stage)] {
			return ""
		}
		seenStages[stageID(stage)] = true

		// search args
		if considerArg {
			for i := len(stage.Args) - 1; i >= 0; i-- {
				if stage.Args[i].Key != variable || stage.Args[i].Line >= untilLine {
					continue
				}

				if buildArgs[stage.Args[i].Key] != "" {
					return strings.Trim(buildArgs[stage.Args[i].Key], "\"'")
				} else if stage.Args[i].Value != "" {
					return strings.Trim(d.replaceVariables(stage.Args[i].Value, buildArgs, baseImageEnv, stage, stage.Args[i].Line), "\"'")
				}
			}
		}

		// search env
		for i := len(stage.Envs) - 1; i >= 0; i-- {
			if stage.Envs[i].Key != variable || stage.Envs[i].Line >= untilLine {
				continue
			}

			if stage.Envs[i].Value != "" {
				return d.replaceVariables(stage.Envs[i].Value, buildArgs, baseImageEnv, stage, stage.Envs[i].Line)
			}
		}

		// is preamble?
		if stage == &d.Preamble.BaseStage {
			if baseImageEnv != nil && baseImageEnv[variable] != "" {
				return baseImageEnv[variable]
			}

			return ""
		}

		// search in preamble
		image := d.replaceVariables(stage.Image, buildArgs, baseImageEnv, &d.Preamble.BaseStage, d.Stages[0].Instructions[0].StartLine)
		foundStage, ok := d.StagesByTarget[image]
		if !ok {
			stage = &d.Preamble.BaseStage
			untilLine = d.Stages[0].Instructions[0].StartLine
			considerArg = true
		} else {
			stage = &foundStage.BaseStage
			untilLine = stage.Instructions[len(stage.Instructions)-1].StartLine + 1
			considerArg = false
		}
	}
}

func RemoveSyntaxVersion(dockerfileContent string) string {
	// just add the syntax and we are done
	return dockerfileSyntax.ReplaceAllString(dockerfileContent, "")
}

func EnsureDockerfileHasFinalStageName(dockerfileContent string, defaultLastStageName string) (string, string, error) {
	result, err := parser.Parse(strings.NewReader(dockerfileContent))
	if err != nil {
		return "", "", err
	}

	// find last from statement
	var lastChild *parser.Node
	for _, child := range result.AST.Children {
		if strings.ToLower(child.Value) == "from" {
			lastChild = child
		}
	}

	// check if there is an AS statement
	if lastChild == nil {
		return "", "", fmt.Errorf("no FROM statement in dockerfile")
	}

	// try to get the last stage
	if lastChild.Next == nil {
		return "", "", fmt.Errorf("cannot parse FROM statement in dockerfile")
	}

	// check next FROM statement
	if lastChild.Next.Next != nil && lastChild.Next.Next.Next != nil && strings.ToLower(lastChild.Next.Next.Value) == "as" {
		return lastChild.Next.Next.Next.Value, "", nil
	}

	// replace FROM statement
	lastChild.Next.Next = &parser.Node{
		Value: "AS",
		Next: &parser.Node{
			Value: defaultLastStageName,
		},
	}
	return defaultLastStageName, ReplaceInDockerfile(dockerfileContent, lastChild), nil
}

func ReplaceInDockerfile(dockerfileContent string, node *parser.Node) string {
	scan := scanner.NewScanner(strings.NewReader(dockerfileContent))

	lines := []string{}
	lineNumber := 0
	for scan.Scan() {
		lineNumber++

		// for now we can only replace
		if lineNumber >= node.StartLine && lineNumber <= node.EndLine {
			lines = append(lines, DumpNode(node))
			continue
		}

		lines = append(lines, scan.Text())
	}

	return strings.Join(lines, "\n")
}

type Dockerfile struct {
	Raw string

	Directives []*parser.Directive
	Preamble   *Preamble
	Syntax     string // https://docs.docker.com/build/concepts/dockerfile/#dockerfile-syntax

	Stages         []*Stage
	StagesByTarget map[string]*Stage
}

type Preamble struct {
	BaseStage
}

type Stage struct {
	BaseStage
	Users []KeyValue
}

type BaseStage struct {
	Image  string
	Target string

	Envs         []KeyValue
	Args         []KeyValue
	Instructions []*parser.Node
}

type KeyValue struct {
	Key   string
	Value string
	Line  int
}

func (d *Dockerfile) Dump() string {
	strs := []string{}
	if d.Preamble != nil {
		strs = append(strs, DumpAll(d.Preamble.Instructions))
	}
	for _, stage := range d.Stages {
		strs = append(strs, DumpAll(stage.Instructions))
	}

	// filter empty strings
	newStrs := []string{}
	for _, str := range strs {
		if str == "" {
			continue
		}

		newStrs = append(newStrs, str)
	}

	return strings.Join(newStrs, "\n")
}

func Parse(dockerfileContent string) (*Dockerfile, error) {
	result, err := parser.Parse(strings.NewReader(dockerfileContent))
	if err != nil {
		return nil, err
	} else if len(result.AST.Children) == 0 {
		return nil, fmt.Errorf("received empty Dockerfile")
	}

	d := &Dockerfile{
		Raw:            dockerfileContent,
		Preamble:       &Preamble{},
		Stages:         nil,
		StagesByTarget: map[string]*Stage{},
	}
	instructions := result.AST.Children

	// parse directives
	directiveParser := parser.DirectiveParser{}
	directives, err := directiveParser.ParseAll([]byte(dockerfileContent))
	if err != nil {
		return nil, err
	}
	d.Directives = directives

	// parse build syntax
	for _, directive := range directives {
		if directive.Name == "syntax" {
			d.Syntax = directive.Value
			break
		}
	}

	// parse instructions
	isPreamble := true
	for _, instruction := range instructions {
		if isPreamble {
			if strings.ToLower(instruction.Value) == "from" {
				isPreamble = false
				d.Stages = append(d.Stages, parseStage(instruction))
				continue
			}

			d.Preamble.Instructions = append(d.Preamble.Instructions, instruction)
			if strings.ToLower(instruction.Value) == "env" {
				d.Preamble.Envs = append(d.Preamble.Envs, parseEnv(instruction)...)
			} else if strings.ToLower(instruction.Value) == "arg" {
				d.Preamble.Args = append(d.Preamble.Args, parseArg(instruction))
			}

			continue
		}

		// is new stage?
		if strings.ToLower(instruction.Value) == "from" {
			d.Stages = append(d.Stages, parseStage(instruction))
			continue
		}

		lastStage := d.Stages[len(d.Stages)-1]
		lastStage.Instructions = append(lastStage.Instructions, instruction)
		if strings.ToLower(instruction.Value) == "env" {
			lastStage.Envs = append(lastStage.Envs, parseEnv(instruction)...)
		} else if strings.ToLower(instruction.Value) == "arg" {
			lastStage.Args = append(lastStage.Args, parseArg(instruction))
		} else if strings.ToLower(instruction.Value) == "user" {
			lastStage.Users = append(lastStage.Users, parseUser(instruction))
		}
	}

	// map stages
	for _, stage := range d.Stages {
		if stage.Target == "" {
			continue
		}

		d.StagesByTarget[stage.Target] = stage
	}

	return d, nil
}

func parseUser(instruction *parser.Node) KeyValue {
	// trim group if necessary
	line := instruction.StartLine
	instruction = instruction.Next
	splitted := strings.Split(instruction.Value, ":")
	return KeyValue{
		Key:  splitted[0],
		Line: line,
	}
}

func parseArg(instruction *parser.Node) KeyValue {
	line := instruction.StartLine
	instruction = instruction.Next
	if instruction.Next != nil {
		return KeyValue{
			Key:   instruction.Value,
			Value: instruction.Next.Value,
			Line:  line,
		}
	}

	if strings.Contains(instruction.Value, "=") {
		splitted := strings.Split(instruction.Value, "=")
		return KeyValue{
			Key:   splitted[0],
			Value: strings.Join(splitted[1:], "="),
			Line:  line,
		}
	}

	return KeyValue{
		Key:  instruction.Value,
		Line: line,
	}
}

func parseEnv(instruction *parser.Node) []KeyValue {
	envs := []KeyValue{}
	line := instruction.StartLine
	node := instruction.Next
	for node != nil {
		if node.Next == nil {
			return envs
		}

		envs = append(envs, KeyValue{
			Key:   strings.TrimSpace(node.Value),
			Value: strings.Trim(strings.ReplaceAll(node.Next.Value, "\\", ""), "\"'"),
			Line:  line,
		})
		node = node.Next.Next
	}

	return envs
}

func parseStage(instruction *parser.Node) *Stage {
	image := ""
	if instruction.Next != nil {
		image = instruction.Next.Value
	}
	target := ""
	if instruction.Next != nil && instruction.Next.Next != nil && strings.ToLower(instruction.Next.Next.Value) == "as" && instruction.Next.Next.Next != nil && instruction.Next.Next.Next.Value != "" {
		target = instruction.Next.Next.Next.Value
	}
	return &Stage{
		BaseStage: BaseStage{
			Image:        image,
			Target:       target,
			Instructions: []*parser.Node{instruction},
		},
	}
}

func DumpAll(result []*parser.Node) string {
	if len(result) == 0 {
		return ""
	}

	children := []string{}
	for _, n := range result {
		children = append(children, DumpNode(n))
	}

	return strings.Join(children, "\n")
}

func DumpNode(node *parser.Node) string {
	out := ""
	if len(node.PrevComment) > 0 {
		out += "# " + strings.Join(node.PrevComment, "\n# ")
	}

	if node.Value != "" {
		if len(node.PrevComment) > 0 {
			out += "\n"
		}

		out += node.Value
	}
	for _, child := range node.Flags {
		out += " " + child
	}
	if node.Next != nil {
		out += " " + DumpNode(node.Next)
	}

	return out
}
