package resolver

import (
	"context"
	"fmt"
	"time"

	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/types"
	"github.com/loft-sh/log/survey"
	"github.com/loft-sh/log/terminal"
	"github.com/pkg/errors"
)

func (r *Resolver) resolveOptions(
	ctx context.Context,
	optionValues map[string]config.OptionValue,
) (map[string]config.OptionValue, error) {
	// copy options
	resolvedOptionValues := map[string]config.OptionValue{}
	for optionName, v := range optionValues {
		resolvedOptionValues[optionName] = v
	}

	// resolve options in reverse order and walk from top to bottom
	for optionNode := r.graph.NextFromTop(); optionNode != nil; optionNode = r.graph.NextFromTop() {
		// resolve next option
		err := r.resolveOption(ctx, optionNode.ID, resolvedOptionValues)
		if err != nil {
			return nil, errors.Wrap(err, "resolve option "+optionNode.ID)
		}

		// resolve sub options
		err = r.refreshSubOptions(ctx, optionNode.ID, resolvedOptionValues)
		if err != nil {
			return nil, fmt.Errorf("refresh sub options for %s: %w", optionNode.ID, err)
		}
	}

	return resolvedOptionValues, nil
}

func (r *Resolver) resolveOption(
	ctx context.Context,
	optionName string,
	resolvedOptionValues map[string]config.OptionValue,
) error {
	// get node from graph
	node := r.graph.Nodes[optionName]
	option := node.Data

	// get existing values
	userValue, userValueOk, beforeValue, beforeValueOk, err := r.getValue(optionName, option, resolvedOptionValues)
	if err != nil {
		return err
	}

	// find out options we need to resolve
	if !userValueOk {
		// check if value is already filled
		if beforeValueOk {
			if beforeValue.UserProvided || option.Cache == "" {
				return nil
			} else if option.Cache != "" {
				duration, err := time.ParseDuration(option.Cache)
				if err != nil {
					return errors.Wrapf(err, "parse cache duration of option %s", optionName)
				}

				// has value expired?
				if beforeValue.Filled != nil && beforeValue.Filled.Add(duration).After(time.Now()) {
					return nil
				}
			}
		}

		// make sure required is always resolved
		if !option.Required {
			// skip if global
			if !r.resolveGlobal && option.Global {
				return nil
			} else if !r.resolveLocal && option.Local {
				return nil
			}
		}
	}

	// resolve option
	if userValueOk {
		resolvedOptionValues[optionName] = config.OptionValue{
			Value:        userValue,
			Children:     beforeValue.Children,
			UserProvided: true,
		}
	} else if option.Default != "" {
		resolvedOptionValues[optionName] = config.OptionValue{
			Children: beforeValue.Children,
			Value:    ResolveDefaultValue(option.Default, combine(resolvedOptionValues, r.extraValues)),
		}
	} else if option.Command != "" {
		optionValue, err := resolveFromCommand(ctx, option, resolvedOptionValues, r.extraValues)
		if err != nil {
			return err
		}

		optionValue.Children = beforeValue.Children
		resolvedOptionValues[optionName] = optionValue
	} else if len(option.Enum) == 1 {
		resolvedOptionValues[optionName] = config.OptionValue{
			Children: beforeValue.Children,
			Value:    option.Enum[0],
		}
	} else {
		resolvedOptionValues[optionName] = config.OptionValue{
			Children: beforeValue.Children,
		}
	}

	// is required?
	if !userValueOk && option.Required && resolvedOptionValues[optionName].Value == "" && !resolvedOptionValues[optionName].UserProvided {
		if r.skipRequired {
			delete(resolvedOptionValues, optionName)
			return deleteChildrenOf(r.graph, node)
		}

		// check if we can ask a question
		if !terminal.IsTerminalIn {
			return fmt.Errorf("option %s is required, but no value provided", optionName)
		}

		// check if there is only one option
		r.log.Info(option.Description)
		answer, err := r.log.Question(&survey.QuestionOptions{
			Question:               fmt.Sprintf("Please enter a value for %s", optionName),
			Options:                option.Enum,
			ValidationRegexPattern: option.ValidationPattern,
			ValidationMessage:      option.ValidationMessage,
			IsPassword:             option.Password,
		})
		if err != nil {
			return err
		}

		resolvedOptionValues[optionName] = config.OptionValue{
			Value:        answer,
			UserProvided: true,
		}
	}

	// check if value has changed
	if beforeValue.Value != resolvedOptionValues[optionName].Value {
		// resolve children again
		for _, child := range node.Childs {
			// check if value is already there
			optionValue, ok := resolvedOptionValues[child.ID]
			if ok && !optionValue.UserProvided {
				// recompute children
				delete(resolvedOptionValues, child.ID)
			}
		}
	}

	return nil
}

func (r *Resolver) getValue(optionName string, option *types.Option, resolvedOptionValues map[string]config.OptionValue) (string, bool, config.OptionValue, bool, error) {
	// check if user value exists
	userValue, userValueOk := r.userOptions[optionName]

	// get before value
	beforeValue, beforeValueOk := resolvedOptionValues[optionName]

	// validate user value if we have one
	if userValueOk {
		err := validateUserValue(optionName, userValue, option)
		if err != nil {
			return "", false, config.OptionValue{}, false, err
		}
	}

	// validate existing value
	if beforeValueOk {
		err := validateUserValue(optionName, beforeValue.Value, option)
		if err != nil {
			// strip before value
			delete(resolvedOptionValues, optionName)
			beforeValue = config.OptionValue{}
			beforeValueOk = false
		}
	}

	return userValue, userValueOk, beforeValue, beforeValueOk, nil
}

func (r *Resolver) refreshSubOptions(
	ctx context.Context,
	optionName string,
	resolvedOptionValues map[string]config.OptionValue,
) error {
	// get options
	node, ok := r.graph.Nodes[optionName]
	if !ok {
		return nil
	}

	// re-fetch dynamic options
	option := node.Data
	if !r.resolveSubOptions || option.SubOptionsCommand == "" {
		return nil
	}

	// only refetch if the option was resolved
	_, ok = resolvedOptionValues[optionName]
	if !ok {
		return nil
	}

	// execute the command
	newDynamicOptions, err := resolveSubOptions(ctx, option, resolvedOptionValues, r.extraValues)
	if err != nil {
		return err
	}

	// remove before children from graph
	for childID := range r.getChangedOptions(r.dynamicOptionsForNode(resolvedOptionValues[optionName].Children), newDynamicOptions, resolvedOptionValues) {
		delete(resolvedOptionValues, childID)
		err := r.graph.RemoveSubGraph(childID)
		if err != nil {
			return err
		}
	}

	// set children on value
	val := resolvedOptionValues[optionName]
	val.Children = []string{}
	for k := range newDynamicOptions {
		val.Children = append(val.Children, k)
	}
	resolvedOptionValues[optionName] = val

	// add options to graph
	err = addOptionsToGraph(r.graph, newDynamicOptions, resolvedOptionValues)
	if err != nil {
		return fmt.Errorf("add sub options: %w", err)
	}

	return nil
}

func (r *Resolver) getChangedOptions(oldOptions config.OptionDefinitions, newOptions config.OptionDefinitions, resolvedOptionValues map[string]config.OptionValue) config.OptionDefinitions {
	changedOptions := config.OptionDefinitions{}
	for oldK, oldV := range oldOptions {
		_, ok := newOptions[oldK]
		if !ok {
			changedOptions[oldK] = oldV
			continue
		}
	}

	for newK, newV := range newOptions {
		oldV, ok := oldOptions[newK]
		if !ok {
			changedOptions[newK] = newV
			continue
		}

		oldValue, oldValueOk := resolvedOptionValues[newK]
		if !oldValueOk {
			changedOptions[newK] = newV
			continue
		}

		// check if value still valid
		if len(newV.Enum) > 0 && !contains(newV.Enum, oldValue.Value) {
			changedOptions[newK] = newV
			continue
		}

		// check if default has changed
		if !oldValue.UserProvided && oldV.Default != newV.Default {
			changedOptions[newK] = newV
			continue
		}
	}

	return changedOptions
}

func (r *Resolver) dynamicOptionsForNode(children []string) config.OptionDefinitions {
	retValues := config.OptionDefinitions{}
	for _, childID := range children {
		child, ok := r.graph.Nodes[childID]
		if ok {
			retValues[child.ID] = child.Data
		}
	}

	return retValues
}

func contains(stack []string, k string) bool {
	for _, s := range stack {
		if s == k {
			return true
		}
	}
	return false
}
