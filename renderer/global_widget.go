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
	Summary       *WorkspaceResource      `json:"summary,omitempty"`
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
	if workspace.Summary != nil {
		if err := workspace.Summary.Validate("summary"); err != nil {
			return err
		}
	}
	if err := workspace.Detail.Validate("detail"); err != nil {
		return err
	}
	if !workspace.Detail.hasSelectionBinding(workspace.Selection.Field) {
		return fmt.Errorf("detail must bind selection field %q", workspace.Selection.Field)
	}
	seenSubscriptions := make(map[string]struct{}, len(workspace.Subscriptions))
	for index, subscription := range workspace.Subscriptions {
		if err := subscription.Validate(); err != nil {
			return fmt.Errorf("subscription %d: %w", index, err)
		}
		key := subscription.Module + "\x00" + strings.Join(subscription.Actions, "\x00") + "\x00" + subscription.Correlation.EventField
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

// ActionResource is a reference to an existing standard module action.
// Generator owns its request and response contracts.
type ActionResource struct {
	Module string `json:"module"`
	Action string `json:"action"`
}

func (resource ActionResource) Validate(name string) error {
	if resource.Module == "" {
		return fmt.Errorf("%s module is required", name)
	}
	if resource.Action == "" {
		return fmt.Errorf("%s action is required", name)
	}
	return nil
}

// WorkspaceResource adds request bindings to an ActionResource. URL, method,
// paging, sorting and response presentation remain owned by the action.
type WorkspaceResource struct {
	ActionResource
	Bindings []WidgetRequestBinding `json:"bindings,omitempty"`
}

func (resource WorkspaceResource) Validate(name string) error {
	if err := resource.ActionResource.Validate(name); err != nil {
		return err
	}
	return ValidateWidgetRequestBindings(resource.Bindings)
}

func (resource WorkspaceResource) hasSelectionBinding(field string) bool {
	for _, binding := range resource.Bindings {
		if binding.Source.Runtime != nil &&
			binding.Source.Runtime.Scope == WidgetRuntimeValueSourceSelection &&
			binding.Source.Runtime.Field == field {
			return true
		}
	}
	return false
}

type WidgetRequestBindingTarget string

const (
	WidgetRequestBindingPathByKey WidgetRequestBindingTarget = "path_by_key"
	WidgetRequestBindingPathValue WidgetRequestBindingTarget = "path_value"
	WidgetRequestBindingFilter    WidgetRequestBindingTarget = "filter"
)

// WidgetRequestBinding applies a typed literal or a known runtime value to a
// generated request. The target determines the request parameter name:
// path_by_key and path_value map to view placeholders, while filter derives
// filter[field] from Field.
type WidgetRequestBinding struct {
	Target WidgetRequestBindingTarget `json:"target"`
	Field  string                     `json:"field,omitempty"`
	Source WidgetValueSource          `json:"source"`
}

type WidgetRuntimeValueSource string

const (
	WidgetRuntimeValueSourceCurrentUser WidgetRuntimeValueSource = "current_user"
	WidgetRuntimeValueSourceSelection   WidgetRuntimeValueSource = "selection"
)

type WidgetRuntimeValue struct {
	Scope WidgetRuntimeValueSource `json:"scope"`
	Field string                   `json:"field"`
}

// WidgetValueSource is a closed union. Literal values are serialized with
// their type; runtime values can only come from a generator-defined scope.
type WidgetValueSource struct {
	Literal *TypedValue         `json:"literal,omitempty"`
	Runtime *WidgetRuntimeValue `json:"runtime,omitempty"`
}

func (source WidgetValueSource) Validate() error {
	variants := 0
	if source.Literal != nil {
		variants++
		if err := source.Literal.Validate(); err != nil {
			return fmt.Errorf("literal: %w", err)
		}
	}
	if source.Runtime != nil {
		variants++
		if source.Runtime.Field == "" {
			return fmt.Errorf("runtime field is required")
		}
		switch source.Runtime.Scope {
		case WidgetRuntimeValueSourceCurrentUser, WidgetRuntimeValueSourceSelection:
		default:
			return fmt.Errorf("runtime scope %q is unsupported", source.Runtime.Scope)
		}
	}
	if variants != 1 {
		return fmt.Errorf("must contain exactly one of literal or runtime")
	}
	return nil
}

func ValidateWidgetRequestBindings(bindings []WidgetRequestBinding) error {
	seen := make(map[string]struct{}, len(bindings))
	for index, binding := range bindings {
		switch binding.Target {
		case WidgetRequestBindingPathByKey, WidgetRequestBindingPathValue:
			if binding.Field != "" {
				return fmt.Errorf("binding %d %s must not define field", index, binding.Target)
			}
		case WidgetRequestBindingFilter:
			if binding.Field == "" {
				return fmt.Errorf("binding %d filter field is required", index)
			}
		default:
			return fmt.Errorf("binding %d has unsupported target %q", index, binding.Target)
		}
		if err := binding.Source.Validate(); err != nil {
			return fmt.Errorf("binding %d source: %w", index, err)
		}
		key := string(binding.Target) + "\x00" + binding.Field
		if _, exists := seen[key]; exists {
			return fmt.Errorf("binding %d duplicates %s %q", index, binding.Target, binding.Field)
		}
		seen[key] = struct{}{}
	}
	return nil
}

type WorkspaceRefreshTarget string

const (
	WorkspaceRefreshSummary WorkspaceRefreshTarget = "summary"
	WorkspaceRefreshMaster  WorkspaceRefreshTarget = "master"
	WorkspaceRefreshDetail  WorkspaceRefreshTarget = "detail"
)

func ValidateWorkspaceRefreshTargets(targets []WorkspaceRefreshTarget) error {
	seen := make(map[WorkspaceRefreshTarget]struct{}, len(targets))
	for index, target := range targets {
		switch target {
		case WorkspaceRefreshSummary, WorkspaceRefreshMaster, WorkspaceRefreshDetail:
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
	Module      string                      `json:"module"`
	Actions     []string                    `json:"actions"`
	Correlation WorkspaceCorrelationBinding `json:"correlation"`
	Refresh     []WorkspaceRefreshTarget    `json:"refresh"`
}

// WorkspaceCorrelationBinding identifies a declared realtime event field. The
// workspace selection is the implicit target and is checked by the generator.
type WorkspaceCorrelationBinding struct {
	EventField string `json:"event_field"`
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
	if subscription.Correlation.EventField == "" {
		return fmt.Errorf("correlation event field is required")
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
	ID        string                        `json:"id"`
	State     WidgetTargetState             `json:"state"`
	Selection *WidgetSelectionResultBinding `json:"selection,omitempty"`
	Refresh   []WorkspaceRefreshTarget      `json:"refresh,omitempty"`
}

// ActionResultField identifies a scalar field declared by a standard action
// result contract.
type ActionResultField string

const (
	ActionResultFieldValue      ActionResultField = "value"
	ActionResultFieldPrimaryKey ActionResultField = "primary_key"
	ActionResultFieldDelete     ActionResultField = "delete"
)

// ActionResultSource identifies a typed scalar field from a standard action
// response. Generator resolves Resource and verifies Field against the action
// result contract.
type ActionResultSource struct {
	Resource ActionResource    `json:"resource"`
	Field    ActionResultField `json:"field"`
}

func (source ActionResultSource) Validate() error {
	if err := source.Resource.Validate("source resource"); err != nil {
		return err
	}
	if source.Field == "" {
		return fmt.Errorf("source field is required")
	}
	return nil
}

// WidgetSelectionResultBinding reads a typed source from the successful
// standard action response. The widget selection field is declared only by
// the target workspace, so source and target cannot be confused.
type WidgetSelectionResultBinding struct {
	Source ActionResultSource `json:"source"`
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
	if target.Selection != nil {
		if err := target.Selection.Source.Validate(); err != nil {
			return fmt.Errorf("selection: %w", err)
		}
	}
	if target.State == WidgetTargetClose && target.Selection != nil {
		return fmt.Errorf("closed widget cannot set selection")
	}
	return ValidateWorkspaceRefreshTargets(target.Refresh)
}

// Actions returns every action declared by the current renderer tree. The
// returned values are detached copies and retain their native result fields.
func (render Universal) Actions() []Action {
	var result []Action
	appendAction := func(action Action) {
		result = append(result, cloneActionValue(action))
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
	return result
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
	Summary  *WidgetResourceLoad `json:"summary,omitempty"`
	Master   *WidgetResourceLoad `json:"master,omitempty"`
	Detail   *WidgetResourceLoad `json:"detail,omitempty"`
}

func cloneWorkspaceWidget(value *WorkspaceWidget) *WorkspaceWidget {
	if value == nil {
		return nil
	}
	cloned := *value
	if value.Summary != nil {
		summary := *value.Summary
		summary.Bindings = cloneWidgetRequestBindings(value.Summary.Bindings)
		cloned.Summary = &summary
	}
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
	for index, value := range values {
		cloned[index] = value
		cloned[index].Source = cloneWidgetValueSource(value.Source)
	}
	return cloned
}

func cloneWidgetValueSource(value WidgetValueSource) WidgetValueSource {
	cloned := value
	if value.Literal != nil {
		literal := *value.Literal
		if value.Literal.Bool != nil {
			boolValue := *value.Literal.Bool
			literal.Bool = &boolValue
		}
		cloned.Literal = &literal
	}
	if value.Runtime != nil {
		runtime := *value.Runtime
		cloned.Runtime = &runtime
	}
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
	if value.Selection != nil {
		selection := *value.Selection
		cloned.Selection = &selection
	}
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
		Summary:  cloneWidgetResourceLoad(value.Summary),
		Master:   cloneWidgetResourceLoad(value.Master),
		Detail:   cloneWidgetResourceLoad(value.Detail),
	}
}
