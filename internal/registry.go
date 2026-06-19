package internal

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/GoCodeAlone/workflow-plugin-admin/internal/contracts"
	"google.golang.org/protobuf/proto"
)

var contributionIDPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)

type contributionRegistry struct {
	mu            sync.RWMutex
	contributions map[string]*contracts.AdminContribution
}

type contributionListRequest struct {
	AppContext          string
	SelectedContextKind string
	SelectedContextID   string
	ContextAuthorized   bool
	GrantedPermissions  []string
}

func newContributionRegistry() *contributionRegistry {
	return &contributionRegistry{contributions: make(map[string]*contracts.AdminContribution)}
}

func (r *contributionRegistry) register(contribution *contracts.AdminContribution) error {
	if contribution == nil {
		return fmt.Errorf("admin contribution is required")
	}
	if !contributionIDPattern.MatchString(contribution.Id) {
		return fmt.Errorf("admin contribution id %q must match %s", contribution.Id, contributionIDPattern.String())
	}
	if strings.TrimSpace(contribution.Title) == "" {
		return fmt.Errorf("admin contribution %q title is required", contribution.Id)
	}
	if !strings.HasPrefix(contribution.Path, "/") {
		return fmt.Errorf("admin contribution %q path must start with /", contribution.Id)
	}
	if contribution.RenderMode == "" {
		contribution.RenderMode = "internal"
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.contributions[contribution.Id] = proto.Clone(contribution).(*contracts.AdminContribution)
	return nil
}

func (r *contributionRegistry) list(appContext string) []*contracts.AdminContribution {
	return r.listForPermissions(appContext, nil)
}

func (r *contributionRegistry) listForPermissions(appContext string, grantedPermissions []string) []*contracts.AdminContribution {
	return r.listForRequest(contributionListRequest{AppContext: appContext, GrantedPermissions: grantedPermissions})
}

func (r *contributionRegistry) listForRequest(request contributionListRequest) []*contracts.AdminContribution {
	r.mu.RLock()
	snapshot := make([]*contracts.AdminContribution, 0, len(r.contributions))
	for _, contribution := range r.contributions {
		snapshot = append(snapshot, contribution)
	}
	r.mu.RUnlock()
	return filterContributionList(snapshot, request)
}

func filterContributionList(contributions []*contracts.AdminContribution, request contributionListRequest) []*contracts.AdminContribution {
	grants := make(map[string]struct{}, len(request.GrantedPermissions))
	for _, permission := range request.GrantedPermissions {
		grants[permission] = struct{}{}
	}

	out := make([]*contracts.AdminContribution, 0, len(contributions))
	for _, contribution := range contributions {
		if contribution == nil {
			continue
		}
		if request.AppContext != "" && contribution.AppContext != "" && contribution.AppContext != request.AppContext {
			continue
		}
		if !contextSelectorVisible(contribution.GetContextSelector(), request, grants) {
			continue
		}
		if !contributionVisible(contribution, grants) {
			continue
		}
		out = append(out, proto.Clone(contribution).(*contracts.AdminContribution))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category == out[j].Category {
			return out[i].Id < out[j].Id
		}
		return out[i].Category < out[j].Category
	})
	return out
}

func contextSelectorVisible(selector *contracts.AdminContextSelector, request contributionListRequest, grants map[string]struct{}) bool {
	if selector == nil {
		return true
	}
	if !request.ContextAuthorized {
		return false
	}
	if selector.GetSelectedContextKey() != "" && request.SelectedContextID == "" {
		return false
	}
	allowedKinds := selector.GetAllowedContextKinds()
	if len(allowedKinds) > 0 {
		if request.SelectedContextKind == "" {
			return false
		}
		found := false
		for _, kind := range allowedKinds {
			if kind == request.SelectedContextKind {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(selector.GetSwitchPermissions()) > 0 && !permissionsVisible(selector.GetSwitchPermissions(), grants) {
		return false
	}
	return true
}

func contributionVisible(contribution *contracts.AdminContribution, grants map[string]struct{}) bool {
	return permissionsVisible(contribution.GetPermissions(), grants)
}

func permissionsVisible(permissions []*contracts.AdminPermission, grants map[string]struct{}) bool {
	if len(permissions) == 0 {
		return true
	}
	for _, permission := range permissions {
		if _, ok := grants[permission.GetPermission()]; ok && permission.GetPermission() != "" {
			return true
		}
		composed := permission.GetResource()
		if permission.GetAction() != "" {
			composed += ":" + permission.GetAction()
		}
		if _, ok := grants[composed]; ok && composed != "" {
			return true
		}
	}
	return false
}
