// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package bootstrap

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/absmach/magistrala/pkg/errors"
)

var errBindingSlot = errors.New("invalid binding slot")

func validateProfileBindingSlots(profile Profile) error {
	seen := make(map[string]struct{}, len(profile.BindingSlots))
	for _, slot := range profile.BindingSlots {
		if slot.Name == "" {
			return fmt.Errorf("%w: slot name is required", errBindingSlot)
		}
		if slot.Type == "" {
			return fmt.Errorf("%w: slot %q type is required", errBindingSlot, slot.Name)
		}
		if _, ok := seen[slot.Name]; ok {
			return fmt.Errorf("%w: duplicate slot %q", errBindingSlot, slot.Name)
		}
		seen[slot.Name] = struct{}{}
	}
	return nil
}

func validateRequestedBindings(profile Profile, requested []BindingRequest) error {
	slots := make(map[string]BindingSlot, len(profile.BindingSlots))
	available := make([]string, 0, len(profile.BindingSlots))
	for _, slot := range profile.BindingSlots {
		slots[slot.Name] = slot
		available = append(available, slot.Name)
	}

	seen := make(map[string]struct{}, len(requested))
	for _, binding := range requested {
		slot, ok := slots[binding.Slot]
		if !ok {
			availableMessage := "none"
			if len(available) > 0 {
				availableMessage = strings.Join(available, ", ")
			}
			return errors.NewRequestError(fmt.Sprintf(
				"binding slot %q is not available in the assigned profile; available slots: %s",
				binding.Slot,
				availableMessage,
			))
		}
		if slot.Type != binding.Type {
			return errors.NewRequestError(fmt.Sprintf("binding slot %q requires type %q, got %q", binding.Slot, slot.Type, binding.Type))
		}
		if _, ok := seen[binding.Slot]; ok {
			return errors.NewRequestError(fmt.Sprintf("binding slot %q is included more than once", binding.Slot))
		}
		seen[binding.Slot] = struct{}{}
	}
	return nil
}

func validateRequiredBindings(profile Profile, bindings []BindingSnapshot) error {
	if len(profile.BindingSlots) == 0 {
		return nil
	}

	bound := make(map[string]BindingSnapshot, len(bindings))
	for _, binding := range bindings {
		bound[binding.Slot] = binding
	}

	missing := make([]string, 0, len(profile.BindingSlots))
	for _, slot := range profile.BindingSlots {
		binding, ok := bound[slot.Name]
		if !slot.Required && !ok {
			continue
		}
		if slot.Required && !ok {
			missing = append(missing, slot.Name)
			continue
		}
		if binding.Type != slot.Type {
			return fmt.Errorf("%w: slot %q expects %q, got %q", errBindingSlot, slot.Name, slot.Type, binding.Type)
		}
	}
	if len(missing) > 0 {
		return errors.NewServiceError(fmt.Sprintf("required binding slots are not bound: %s", strings.Join(missing, ", ")))
	}
	return nil
}

func mergeBindingSnapshots(existing, updated []BindingSnapshot) []BindingSnapshot {
	merged := make(map[string]BindingSnapshot, len(existing)+len(updated))
	for _, binding := range existing {
		merged[binding.Slot] = binding
	}
	for _, binding := range updated {
		merged[binding.Slot] = binding
	}

	bindings := make([]BindingSnapshot, 0, len(merged))
	for _, binding := range merged {
		bindings = append(bindings, binding)
	}
	return bindings
}

func validateProfileTemplate(p Profile) error {
	if p.ContentTemplate == "" || p.ContentFormat == ContentFormatRaw {
		return nil
	}
	_, err := template.New("bootstrap").Funcs(allowlistedFuncs()).Parse(p.ContentTemplate)
	if err != nil {
		return errors.Wrap(ErrRenderFailed, err)
	}
	return nil
}
