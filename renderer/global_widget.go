package renderer

import (
	"fmt"
	"strings"
)

// GlobalWidget describes a shell-level UI surface. It does not create a route
// and can either load the action that registered it or compose a workspace of
// referenced module actions.
type GlobalWidget struct {
	Surface   WidgetSurface    `json:"surface"`
	Workspace *WorkspaceWidget `json:"workspace,omitempty"`
}

func (widget GlobalWidget) Identity() Identity {
	return UniversalIdentity()
}

func (widget GlobalWidget) Clone() GlobalWidget {
	cloned := widget
	cloned.Workspace = cloneWorkspaceWidget(widget.Workspace)
	return cloned
}

// LocalizeGlobalWidget mirrors the renderer localization entry point. The
// current widget contract deliberately has no copy fields: all user-visible
// text belongs to the referenced action responses.
func LocalizeGlobalWidget(widget GlobalWidget, _ TextResolver) GlobalWidget {
	return widget.Clone()
}

func (widget GlobalWidget) Validate() error {
	if err := widget.Surface.Validate(); err != nil {
		return fmt.Errorf("renderer.GlobalWidget: surface: %w", err)
	}
	if widget.Workspace != nil {
		if err := widget.Workspace.Validate(); err != nil {
			return fmt.Errorf("renderer.GlobalWidget: workspace: %w", err)
		}
	}
	return nil
}

type WidgetSurfaceKind string

const (
	WidgetSurfaceDrawer WidgetSurfaceKind = "drawer"
	WidgetSurfacePopup  WidgetSurfaceKind = "popup"
)

type WidgetPlacement string

const (
	WidgetPlacementShellStart WidgetPlacement = "shell_start"
	WidgetPlacementShellEnd   WidgetPlacement = "shell_end"
	WidgetPlacementCenter     WidgetPlacement = "center"
)

type WidgetLoadPolicy string

const (
	WidgetLoadOnOpen WidgetLoadPolicy = "on_open"
	WidgetLoadEager  WidgetLoadPolicy = "eager"
)

type WidgetSurface struct {
	Kind       WidgetSurfaceKind `json:"kind"`
	Placement  WidgetPlacement   `json:"placement"`
	LoadPolicy WidgetLoadPolicy  `json:"load_policy"`
}

func (surface WidgetSurface) Validate() error {
	switch surface.Kind {
	case WidgetSurfaceDrawer, WidgetSurfacePopup:
	default:
		return fmt.Errorf("unsupported kind %q", surface.Kind)
	}
	switch surface.Placement {
	case WidgetPlacementShellStart, WidgetPlacementShellEnd, WidgetPlacementCenter:
	default:
		return fmt.Errorf("unsupported placement %q", surface.Placement)
	}
	if surface.Kind == WidgetSurfaceDrawer && surface.Placement == WidgetPlacementCenter {
		return fmt.Errorf("drawer does not support placement %q", surface.Placement)
	}
	switch surface.LoadPolicy {
	case WidgetLoadOnOpen, WidgetLoadEager:
	default:
		return fmt.Errorf("unsupported load policy %q", surface.LoadPolicy)
	}
	return nil
}

// WorkspaceWidget composes server resources into a generic master-detail
// shell surface. Resources remain normal module actions.
type WorkspaceWidget struct {
	Selection     WorkspaceSelection      `json:"selection"`
	Master        WorkspaceResource       `json:"master"`
	Detail        WorkspaceResource       `json:"detail"`
	Subscriptions []WorkspaceSubscription `json:"subscriptions,omitempty"`
}

func (workspace WorkspaceWidget) Validate() error {
	if workspace.Selection.Field == "" {
		return fmt.Errorf("selection field is required")
	}
	if err := workspace.Master.Validate("master"); err != nil {
		return err
	}
	if err := workspace.Detail.Validate("detail"); err != nil {
		return err
	}
	if !workspace.Detail.hasSelectionBinding(workspace.Selection.Field) {
		return fmt.Errorf("detail must bind selection.%s", workspace.Selection.Field)
	}
	seenSubscriptions := make(map[string]struct{}, len(workspace.Subscriptions))
	for index, subscription := range workspace.Subscriptions {
		if err := subscription.Validate(); err != nil {
			return fmt.Errorf("subscription %d: %w", index, err)
		}
		key := subscription.Module + "\x00" + strings.Join(subscription.Actions, "\x00") + "\x00" + subscription.CorrelationKey
		if _, exists := seenSubscriptions[key]; exists {
			return fmt.Errorf("subscription %d is duplicated", index)
		}
		seenSubscriptions[key] = struct{}{}
	}
	return nil
}

type WorkspaceSelection struct {
	Field string `json:"field"`
}

// WorkspaceResource references an existing module action. Bindings only carry
// runtime values; URL, method, paging, sorting and response presentation are
// resolved from the referenced action by the generator.
type WorkspaceResource struct {
	Module   string                 `json:"module"`
	Action   string                 `json:"action"`
	Bindings []WidgetRequestBinding `json:"bindings,omitempty"`
}

func (resource WorkspaceResource) Validate(name string) error {
	if resource.Module == "" {
		return fmt.Errorf("%s module is required", name)
	}
	if resource.Action == "" {
		return fmt.Errorf("%s action is required", name)
	}
	return ValidateWidgetRequestBindings(resource.Bindings)
}

func (resource WorkspaceResource) hasSelectionBinding(field string) bool {
	for _, binding := range resource.Bindings {
		if binding.Value == "selection."+field {
			return true
		}
	}
	return false
}

type WidgetRequestBindingTarget string

const (
	WidgetRequestBindingPath  WidgetRequestBindingTarget = "path"
	WidgetRequestBindingQuery WidgetRequestBindingTarget = "query"
)

// WidgetRequestBinding applies a static value or a declared runtime scope such
// as selection.id to a generated request.
type WidgetRequestBinding struct {
	Target WidgetRequestBindingTarget `json:"target"`
	Name   string                     `json:"name"`
	Value  string                     `json:"value"`
}

func ValidateWidgetRequestBindings(bindings []WidgetRequestBinding) error {
	seen := make(map[string]struct{}, len(bindings))
	for index, binding := range bindings {
		switch binding.Target {
		case WidgetRequestBindingPath, WidgetRequestBindingQuery:
		default:
			return fmt.Errorf("binding %d has unsupported target %q", index, binding.Target)
		}
		if binding.Name == "" {
			return fmt.Errorf("binding %d name is required", index)
		}
		if binding.Value == "" {
			return fmt.Errorf("binding %d value is required", index)
		}
		key := string(binding.Target) + "\x00" + binding.Name
		if _, exists := seen[key]; exists {
			return fmt.Errorf("binding %d duplicates %s %q", index, binding.Target, binding.Name)
		}
		seen[key] = struct{}{}
	}
	return nil
}

type WorkspaceRefreshTarget string

const (
	WorkspaceRefreshMaster WorkspaceRefreshTarget = "master"
	WorkspaceRefreshDetail WorkspaceRefreshTarget = "detail"
)

func ValidateWorkspaceRefreshTargets(targets []WorkspaceRefreshTarget) error {
	seen := make(map[WorkspaceRefreshTarget]struct{}, len(targets))
	for index, target := range targets {
		switch target {
		case WorkspaceRefreshMaster, WorkspaceRefreshDetail:
		default:
			return fmt.Errorf("refresh target %d is unsupported: %q", index, target)
		}
		if _, exists := seen[target]; exists {
			return fmt.Errorf("refresh target %q is duplicated", target)
		}
		seen[target] = struct{}{}
	}
	return nil
}

type WorkspaceSubscription struct {
	Module         string                   `json:"module"`
	Actions        []string                 `json:"actions"`
	CorrelationKey string                   `json:"correlation_key"`
	Refresh        []WorkspaceRefreshTarget `json:"refresh"`
}

func (subscription WorkspaceSubscription) Validate() error {
	if subscription.Module == "" {
		return fmt.Errorf("module is required")
	}
	if len(subscription.Actions) == 0 {
		return fmt.Errorf("actions are required")
	}
	seenActions := make(map[string]struct{}, len(subscription.Actions))
	for _, action := range subscription.Actions {
		if action == "" {
			return fmt.Errorf("action is required")
		}
		if _, exists := seenActions[action]; exists {
			return fmt.Errorf("action %q is duplicated", action)
		}
		seenActions[action] = struct{}{}
	}
	if subscription.CorrelationKey == "" {
		return fmt.Errorf("correlation key is required")
	}
	if len(subscription.Refresh) == 0 {
		return fmt.Errorf("refresh targets are required")
	}
	return ValidateWorkspaceRefreshTargets(subscription.Refresh)
}

type WidgetTargetState string

const (
	WidgetTargetOpen  WidgetTargetState = "open"
	WidgetTargetClose WidgetTargetState = "close"
)

// WidgetTarget is the typed result of a standard action that controls a
// registered global widget.
type WidgetTarget struct {
	ID             string                   `json:"id"`
	State          WidgetTargetState        `json:"state"`
	SelectionField string                   `json:"selection_field,omitempty"`
	Refresh        []WorkspaceRefreshTarget `json:"refresh,omitempty"`
}

func (target WidgetTarget) Validate() error {
	if target.ID == "" {
		return fmt.Errorf("widget id is required")
	}
	switch target.State {
	case WidgetTargetOpen, WidgetTargetClose:
	default:
		return fmt.Errorf("unsupported widget state %q", target.State)
	}
	if target.State == WidgetTargetClose && target.SelectionField != "" {
		return fmt.Errorf("closed widget cannot set selection")
	}
	return ValidateWorkspaceRefreshTargets(target.Refresh)
}

// WidgetTargets returns every typed widget result declared by the existing
// page structures. Generator uses it to validate target IDs against registered
// global widgets without adding module-specific branches to renderers.
func (render Universal) WidgetTargets() []WidgetTarget {
	var targets []WidgetTarget
	appendAction := func(action Action) {
		if action.AfterSuccess != nil && action.AfterSuccess.Widget != nil {
			targets = append(targets, *cloneWidgetTarget(action.AfterSuccess.Widget))
		}
		if action.AfterError != nil && action.AfterError.Widget != nil {
			targets = append(targets, *cloneWidgetTarget(action.AfterError.Widget))
		}
	}
	appendActions := func(actions []Action) {
		for _, action := range actions {
			appendAction(action)
		}
	}
	appendList := func(page *ListPage) {
		if page == nil {
			return
		}
		appendActions(page.Actions)
		if page.CardSchema != nil {
			appendActions(page.CardSchema.Actions)
		}
	}
	appendList(render.List)
	if render.Form != nil {
		appendActions(render.Form.Actions)
		for _, section := range render.Form.Sections {
			appendList(section.ListPage)
			if section.Collection != nil {
				appendActions(section.Collection.Actions)
				for _, bucket := range section.Collection.Buckets {
					appendActions(bucket.Actions)
				}
			}
			if section.MediaActions != nil {
				for _, action := range []*Action{
					section.MediaActions.Upload,
					section.MediaActions.Link,
					section.MediaActions.Reorder,
					section.MediaActions.Recenter,
					section.MediaActions.Crop,
					section.MediaActions.Remove,
				} {
					if action != nil {
						appendAction(*action)
					}
				}
			}
		}
	}
	if render.Record != nil {
		appendActions(render.Record.Actions)
	}
	if render.ResourceGrid != nil {
		for _, action := range []*Action{render.ResourceGrid.Create, render.ResourceGrid.Delete, render.ResourceGrid.Update} {
			if action != nil {
				appendAction(*action)
			}
		}
		if render.ResourceGrid.Card != nil {
			appendActions(render.ResourceGrid.Card.Actions)
		}
	}
	return targets
}

// WidgetResourceLoad is the resolved request for one global widget resource.
// The response itself provides the existing ListPage, RecordPage or FormPage
// presentation metadata.
type WidgetResourceLoad struct {
	Request  APIAction              `json:"request"`
	Bindings []WidgetRequestBinding `json:"bindings,omitempty"`
}

type WidgetLoad struct {
	Resource *WidgetResourceLoad `json:"resource,omitempty"`
	Master   *WidgetResourceLoad `json:"master,omitempty"`
	Detail   *WidgetResourceLoad `json:"detail,omitempty"`
}

func cloneWorkspaceWidget(value *WorkspaceWidget) *WorkspaceWidget {
	if value == nil {
		return nil
	}
	cloned := *value
	cloned.Master.Bindings = cloneWidgetRequestBindings(value.Master.Bindings)
	cloned.Detail.Bindings = cloneWidgetRequestBindings(value.Detail.Bindings)
	cloned.Subscriptions = cloneWorkspaceSubscriptions(value.Subscriptions)
	return &cloned
}

func cloneWidgetRequestBindings(values []WidgetRequestBinding) []WidgetRequestBinding {
	if values == nil {
		return nil
	}
	cloned := make([]WidgetRequestBinding, len(values))
	copy(cloned, values)
	return cloned
}

func cloneWorkspaceSubscriptions(values []WorkspaceSubscription) []WorkspaceSubscription {
	if values == nil {
		return nil
	}
	cloned := make([]WorkspaceSubscription, len(values))
	for index, value := range values {
		cloned[index] = value
		cloned[index].Actions = cloneSlice(value.Actions)
		cloned[index].Refresh = cloneSlice(value.Refresh)
	}
	return cloned
}

func cloneWidgetTarget(value *WidgetTarget) *WidgetTarget {
	if value == nil {
		return nil
	}
	cloned := *value
	cloned.Refresh = cloneSlice(value.Refresh)
	return &cloned
}

func cloneWidgetResourceLoad(value *WidgetResourceLoad) *WidgetResourceLoad {
	if value == nil {
		return nil
	}
	cloned := *value
	cloned.Request = *cloneAPIAction(&value.Request)
	cloned.Bindings = cloneWidgetRequestBindings(value.Bindings)
	return &cloned
}

func (value WidgetLoad) Clone() WidgetLoad {
	return WidgetLoad{
		Resource: cloneWidgetResourceLoad(value.Resource),
		Master:   cloneWidgetResourceLoad(value.Master),
		Detail:   cloneWidgetResourceLoad(value.Detail),
	}
}
