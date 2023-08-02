package resolver

import (
	"fmt"
	"sort"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/devcontainer/graph"
	"github.com/loft-sh/devpod/pkg/types"
)

func combine(resolvedOptions map[string]config.OptionValue, extraValues map[string]string) map[string]string {
	options := map[string]string{}
	for k, v := range extraValues {
		options[k] = v
	}
	for k, v := range resolvedOptions {
		options[k] = v.Value
	}
	return options
}

func addDependencies(g *graph.Graph[*types.Option], options config.OptionDefinitions, optionValues map[string]config.OptionValue) error {
	// add options
	for optionName := range options {
		err := addDependency(g, optionValues, optionName)
		if err != nil {
			return err
		}
	}

	// remove root parent if possible
	removeRootParent(g)
	return nil
}

func addDependency(g *graph.Graph[*types.Option], optionValues map[string]config.OptionValue, optionName string) error {
	option := g.Nodes[optionName].Data

	// Always add children as dependencies
	for _, childName := range optionValues[optionName].Children {
		if g.Nodes[childName] == nil || childName == optionName {
			continue
		}

		if !option.Global && g.Nodes[childName].Data.Global {
			return fmt.Errorf("cannot use a global option as a dependency of a non-global option. Option '%s' used in command of option '%s'", childName, optionName)
		} else if option.Local && !g.Nodes[childName].Data.Local {
			return fmt.Errorf("cannot use a non-local option as a dependency of a local option. Option '%s' used in default of option '%s'", childName, optionName)
		}

		err := g.AddChild(optionName, childName)
		if err != nil {
			return err
		}
	}

	// Find variables in default value
	for _, dep := range findVariables(option.Default) {
		if g.Nodes[dep] == nil || dep == optionName {
			continue
		}

		if option.Global && !g.Nodes[dep].Data.Global {
			return fmt.Errorf("cannot use a global option as a dependency of a non-global option. Option '%s' used in default of option '%s'", dep, optionName)
		} else if !option.Local && g.Nodes[dep].Data.Local {
			return fmt.Errorf("cannot use a non-local option as a dependency of a local option. Option '%s' used in default of option '%s'", dep, optionName)
		}

		err := g.AddChild(dep, optionName)
		if err != nil {
			return err
		}
	}

	// Find variables in command value
	for _, dep := range findVariables(option.Command) {
		if g.Nodes[dep] == nil || dep == optionName {
			continue
		}

		if option.Global && !g.Nodes[dep].Data.Global {
			return fmt.Errorf("cannot use a global option as a dependency of a non-global option. Option '%s' used in command of option '%s'", dep, optionName)
		} else if !option.Local && g.Nodes[dep].Data.Local {
			return fmt.Errorf("cannot use a non-local option as a dependency of a local option. Option '%s' used in default of option '%s'", dep, optionName)
		}

		err := g.AddChild(dep, optionName)
		if err != nil {
			return err
		}
	}

	return nil
}

func addOptionsToGraph(g *graph.Graph[*types.Option], optionDefinitions config.OptionDefinitions, optionValues map[string]config.OptionValue) error {
	for optionName, option := range optionDefinitions {
		_, ok := g.Nodes[optionName]
		if ok {
			g.Nodes[optionName].Data = option
			continue
		}

		_, err := g.InsertNodeAt(rootID, optionName, option)
		if err != nil {
			return err
		}
	}

	// add dependencies
	err := addDependencies(g, optionDefinitions, optionValues)
	if err != nil {
		return err
	}

	return nil
}

func deleteChildrenOf(graph *graph.Graph[*types.Option], node *graph.Node[*types.Option]) error {
	for _, child := range node.Childs {
		err := graph.RemoveSubGraph(child.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeRootParent(g *graph.Graph[*types.Option]) {
	for optionName := range g.Nodes {
		node := g.Nodes[optionName]

		// remove root parent
		if len(node.Parents) > 1 {
			newParents := []*graph.Node[*types.Option]{}
			removed := false
			for _, parent := range node.Parents {
				if parent.ID == rootID {
					removed = true
					continue
				}
				newParents = append(newParents, parent)
			}
			node.Parents = newParents

			// remove from root childs
			if removed {
				newChilds := []*graph.Node[*types.Option]{}
				for _, child := range g.Root.Childs {
					if child.ID == node.ID {
						continue
					}
					newChilds = append(newChilds, child)
				}
				g.Root.Childs = newChilds
			}
		}
	}
}

func findVariables(str string) []string {
	retVars := map[string]bool{}
	matches := variableExpression.FindAllStringSubmatch(str, -1)
	for _, match := range matches {
		if len(match) != 5 {
			continue
		}

		retVars[match[1]] = true
	}

	retVarsArr := []string{}
	for k := range retVars {
		retVarsArr = append(retVarsArr, k)
	}

	sort.Strings(retVarsArr)
	return retVarsArr
}

func mergeMaps[K comparable, V any](existing map[K]V, newOpts map[K]V) map[K]V {
	retOpts := map[K]V{}
	for k, v := range existing {
		retOpts[k] = v
	}
	for k, v := range newOpts {
		retOpts[k] = v
	}

	return retOpts
}
