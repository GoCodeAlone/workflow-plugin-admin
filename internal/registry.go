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
	grants := make(map[string]struct{}, len(grantedPermissions))
	for _, permission := range grantedPermissions {
		grants[permission] = struct{}{}
	}

	r.mu.RLock()
	snapshot := make([]*contracts.AdminContribution, 0, len(r.contributions))
	for _, contribution := range r.contributions {
		snapshot = append(snapshot, contribution)
	}
	r.mu.RUnlock()
	return filterContributionList(snapshot, appContext, grants)
}

func filterContributionList(contributions []*contracts.AdminContribution, appContext string, grants map[string]struct{}) []*contracts.AdminContribution {
	out := make([]*contracts.AdminContribution, 0, len(contributions))
	for _, contribution := range contributions {
		if contribution == nil {
			continue
		}
		if appContext != "" && contribution.AppContext != "" && contribution.AppContext != appContext {
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

func contributionVisible(contribution *contracts.AdminContribution, grants map[string]struct{}) bool {
	if len(contribution.GetPermissions()) == 0 {
		return true
	}
	for _, permission := range contribution.GetPermissions() {
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
