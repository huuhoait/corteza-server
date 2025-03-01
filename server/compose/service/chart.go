package service

import (
	"context"
	"github.com/cortezaproject/corteza/server/compose/types"
	"github.com/cortezaproject/corteza/server/pkg/actionlog"
	"github.com/cortezaproject/corteza/server/pkg/errors"
	"github.com/cortezaproject/corteza/server/pkg/handle"
	"github.com/cortezaproject/corteza/server/pkg/label"
	"github.com/cortezaproject/corteza/server/pkg/locale"
	"github.com/cortezaproject/corteza/server/store"
	"reflect"
	"strconv"
)

type (
	chart struct {
		actionlog actionlog.Recorder
		ac        chartAccessController
		store     store.Storer
		locale    ResourceTranslationsManagerService
	}

	chartAccessController interface {
		CanManageResourceTranslations(ctx context.Context) bool
		CanSearchChartsOnNamespace(context.Context, *types.Namespace) bool
		CanReadNamespace(context.Context, *types.Namespace) bool
		CanCreateChartOnNamespace(context.Context, *types.Namespace) bool
		CanReadChart(context.Context, *types.Chart) bool
		CanUpdateChart(context.Context, *types.Chart) bool
		CanDeleteChart(context.Context, *types.Chart) bool
	}

	chartUpdateHandler func(ctx context.Context, ns *types.Namespace, c *types.Chart) (chartChanges, error)

	chartChanges uint8
)

const (
	chartUnchanged     chartChanges = 0
	chartChanged       chartChanges = 1
	chartLabelsChanged chartChanges = 2
)

func Chart() *chart {
	return &chart{
		ac:        DefaultAccessControl,
		actionlog: DefaultActionlog,
		store:     DefaultStore,
		locale:    DefaultResourceTranslation,
	}
}

func (svc chart) Find(ctx context.Context, filter types.ChartFilter) (set types.ChartSet, f types.ChartFilter, err error) {
	var (
		aProps = &chartActionProps{filter: &filter}
		ns     *types.Namespace
	)

	// For each fetched item, store backend will check if it is valid or not
	filter.Check = func(res *types.Chart) (bool, error) {
		if !svc.ac.CanReadChart(ctx, res) {
			return false, nil
		}

		return true, nil
	}

	err = func() error {
		ns, err = loadNamespace(ctx, svc.store, filter.NamespaceID)
		if err != nil {
			return err
		}

		aProps.setNamespace(ns)
		if !svc.ac.CanSearchChartsOnNamespace(ctx, ns) {
			return ChartErrNotAllowedToSearch()
		}

		if len(filter.Labels) > 0 {
			filter.LabeledIDs, err = label.Search(
				ctx,
				svc.store,
				types.Chart{}.LabelResourceKind(),
				filter.Labels,
				filter.ChartID...,
			)

			if err != nil {
				return err
			}

			// labels specified but no labeled resources found
			if len(filter.LabeledIDs) == 0 {
				return nil
			}
		}

		if set, f, err = store.SearchComposeCharts(ctx, svc.store, filter); err != nil {
			return err
		}

		if err = label.Load(ctx, svc.store, toLabeledCharts(set)...); err != nil {
			return err
		}

		set.Walk(func(c *types.Chart) error {
			svc.proc(ctx, c)
			return nil
		})

		return nil
	}()

	return set, f, svc.recordAction(ctx, aProps, ChartActionSearch, err)
}

func (svc chart) FindByID(ctx context.Context, namespaceID, chartID uint64) (c *types.Chart, err error) {
	return svc.lookup(ctx, namespaceID, func(aProps *chartActionProps) (*types.Chart, error) {
		return loadChart(ctx, svc.store, namespaceID, chartID)
	})
}

func (svc chart) FindByHandle(ctx context.Context, namespaceID uint64, h string) (c *types.Chart, err error) {
	return svc.lookup(ctx, namespaceID, func(aProps *chartActionProps) (*types.Chart, error) {
		if !handle.IsValid(h) {
			return nil, ChartErrInvalidHandle()
		}

		aProps.chart.Handle = h
		return store.LookupComposeChartByNamespaceIDHandle(ctx, svc.store, namespaceID, h)
	})
}

func (svc chart) proc(ctx context.Context, c *types.Chart) {
	if svc.locale == nil || svc.locale.Locale() == nil {
		return
	}

	tag := locale.GetAcceptLanguageFromContext(ctx)
	c.DecodeTranslations(svc.locale.Locale().ResourceTranslations(tag, c.ResourceTranslation()))
}

func (svc chart) Create(ctx context.Context, new *types.Chart) (*types.Chart, error) {
	var (
		err    error
		ns     *types.Namespace
		aProps = &chartActionProps{chart: new}
	)

	err = store.Tx(ctx, svc.store, func(ctx context.Context, s store.Storer) (err error) {
		if !handle.IsValid(new.Handle) {
			return ChartErrInvalidHandle()
		}

		if ns, err = loadNamespace(ctx, s, new.NamespaceID); err != nil {
			return err
		}

		aProps.setNamespace(ns)

		if !svc.ac.CanCreateChartOnNamespace(ctx, ns) {
			return ChartErrNotAllowedToCreate()
		}

		if err = svc.uniqueCheck(ctx, new); err != nil {
			return err
		}

		new.ID = nextID()
		new.CreatedAt = *now()
		new.UpdatedAt = nil
		new.DeletedAt = nil

		// generate config element IDs
		new.Config.GenerateIDs(nextID)

		aProps.setChanged(new)

		if err = store.CreateComposeChart(ctx, s, new); err != nil {
			return err
		}

		if err = updateTranslations(ctx, svc.ac, svc.locale, new.EncodeTranslations()...); err != nil {
			return
		}

		if err = label.Create(ctx, s, new); err != nil {
			return
		}

		return nil
	})

	return new, svc.recordAction(ctx, aProps, ChartActionCreate, err)
}

func (svc chart) Update(ctx context.Context, upd *types.Chart) (c *types.Chart, err error) {
	return svc.updater(ctx, upd.NamespaceID, upd.ID, ChartActionUpdate, svc.handleUpdate(ctx, upd))
}

func (svc chart) DeleteByID(ctx context.Context, namespaceID, chartID uint64) error {
	return trim1st(svc.updater(ctx, namespaceID, chartID, ChartActionDelete, svc.handleDelete))
}

func (svc chart) UndeleteByID(ctx context.Context, namespaceID, chartID uint64) error {
	return trim1st(svc.updater(ctx, namespaceID, chartID, ChartActionUndelete, svc.handleUndelete))
}

// lookup fn() orchestrates chart lookup, namespace preload and check
func (svc chart) lookup(ctx context.Context, namespaceID uint64, lookup func(*chartActionProps) (*types.Chart, error)) (c *types.Chart, err error) {
	var aProps = &chartActionProps{chart: &types.Chart{NamespaceID: namespaceID}}

	err = func() error {
		if ns, err := loadNamespace(ctx, svc.store, namespaceID); err != nil {
			return err
		} else {
			aProps.setNamespace(ns)
		}

		if c, err = lookup(aProps); errors.IsNotFound(err) {
			return ChartErrNotFound()
		} else if err != nil {
			return err
		}

		aProps.setChart(c)

		if !svc.ac.CanReadChart(ctx, c) {
			return ChartErrNotAllowedToRead()
		}

		svc.proc(ctx, c)

		if err = label.Load(ctx, svc.store, c); err != nil {
			return err
		}

		return nil
	}()

	return c, svc.recordAction(ctx, aProps, ChartActionLookup, err)
}

func (svc chart) updater(ctx context.Context, namespaceID, chartID uint64, action func(...*chartActionProps) *chartAction, fn chartUpdateHandler) (*types.Chart, error) {
	var (
		changes chartChanges
		ns      *types.Namespace
		c       *types.Chart
		aProps  = &chartActionProps{chart: &types.Chart{ID: chartID, NamespaceID: namespaceID}}
		err     error
	)

	err = store.Tx(ctx, svc.store, func(ctx context.Context, s store.Storer) (err error) {
		ns, c, err = loadChartCombo(ctx, s, namespaceID, chartID)
		if err != nil {
			return
		}

		if err = label.Load(ctx, svc.store, c); err != nil {
			return err
		}

		aProps.setNamespace(ns)
		aProps.setChanged(c)

		if changes, err = fn(ctx, ns, c); err != nil {
			return err
		}

		// generate config element IDs if missing
		c.Config.GenerateIDs(nextID)

		if changes&chartChanged > 0 {
			if err = store.UpdateComposeChart(ctx, s, c); err != nil {
				return err
			}
		}

		if err = updateTranslations(ctx, svc.ac, svc.locale, c.EncodeTranslations()...); err != nil {
			return
		}

		if changes&chartLabelsChanged > 0 {
			if err = label.Update(ctx, s, c); err != nil {
				return
			}
		}

		return nil
	})

	return c, svc.recordAction(ctx, aProps, action, err)
}

func (svc chart) uniqueCheck(ctx context.Context, c *types.Chart) (err error) {
	if c.Handle != "" {
		if e, _ := store.LookupComposeChartByNamespaceIDHandle(ctx, svc.store, c.NamespaceID, c.Handle); e != nil && e.ID != c.ID {
			return ChartErrHandleNotUnique()
		}
	}

	return nil
}

func (svc chart) handleUpdate(ctx context.Context, upd *types.Chart) chartUpdateHandler {
	return func(ctx context.Context, ns *types.Namespace, res *types.Chart) (changes chartChanges, err error) {
		if isStale(upd.UpdatedAt, res.UpdatedAt, res.CreatedAt) {
			return chartUnchanged, ChartErrStaleData()
		}

		if upd.Handle != res.Handle && !handle.IsValid(upd.Handle) {
			return chartUnchanged, ChartErrInvalidHandle()
		}

		if err := svc.uniqueCheck(ctx, upd); err != nil {
			return chartUnchanged, err
		}

		if !svc.ac.CanUpdateChart(ctx, res) {
			return chartUnchanged, ChartErrNotAllowedToUpdate()
		}

		if res.Name != upd.Name {
			changes |= chartChanged
			res.Name = upd.Name
		}

		if res.Handle != upd.Handle {
			changes |= chartChanged
			res.Handle = upd.Handle
		}

		if !reflect.DeepEqual(upd.Config, res.Config) {
			changes |= chartChanged
			res.Config = upd.Config
		}

		// Assure ReportIDs
		for i, r := range res.Config.Reports {
			if r.ReportID == 0 {
				r.ReportID = nextID()
				res.Config.Reports[i] = r

				changes |= chartChanged
			}

			// Ensure chart report metric IDs
			for j, m := range r.Metrics {
				if val, ok := m["metricID"]; !ok || val == 0 {
					m["metricID"] = strconv.FormatUint(nextID(), 10)
					res.Config.Reports[i].Metrics[j] = m

					changes |= chartChanged
				}
			}
		}
		if changes&chartChanged > 0 {
			res.UpdatedAt = now()
		}

		if upd.Labels != nil {
			if label.Changed(res.Labels, upd.Labels) {
				changes |= chartLabelsChanged
				res.Labels = upd.Labels
			}
		}

		return
	}
}

func (svc chart) handleDelete(ctx context.Context, ns *types.Namespace, c *types.Chart) (chartChanges, error) {
	if !svc.ac.CanDeleteChart(ctx, c) {
		return chartUnchanged, ChartErrNotAllowedToDelete()
	}

	if c.DeletedAt != nil {
		// chart already deleted
		return chartUnchanged, nil
	}

	c.DeletedAt = now()
	return chartChanged, nil
}

func (svc chart) handleUndelete(ctx context.Context, ns *types.Namespace, c *types.Chart) (chartChanges, error) {
	if !svc.ac.CanDeleteChart(ctx, c) {
		return chartUnchanged, ChartErrNotAllowedToUndelete()
	}

	if c.DeletedAt == nil {
		// chart not deleted
		return chartUnchanged, nil
	}

	c.DeletedAt = nil
	return chartChanged, nil
}

func loadChartCombo(ctx context.Context, s interface {
	store.ComposeCharts
	store.ComposeNamespaces
}, namespaceID, chartID uint64) (ns *types.Namespace, c *types.Chart, err error) {
	ns, err = loadNamespace(ctx, s, namespaceID)
	if err != nil {
		return
	}

	c, err = loadChart(ctx, s, namespaceID, chartID)
	return
}

func loadChart(ctx context.Context, s store.ComposeCharts, namespaceID, chartID uint64) (res *types.Chart, err error) {
	if chartID == 0 || namespaceID == 0 {
		return nil, ChartErrInvalidID()
	}

	if res, err = store.LookupComposeChartByID(ctx, s, chartID); errors.IsNotFound(err) {
		err = ChartErrNotFound()
	}

	if err == nil && namespaceID != res.NamespaceID {
		// Make sure chart belongs to the right namespace
		return nil, ChartErrNotFound()
	}

	return
}

// toLabeledCharts converts to []label.LabeledResource
//
// This function is auto-generated.
func toLabeledCharts(set []*types.Chart) []label.LabeledResource {
	if len(set) == 0 {
		return nil
	}

	ll := make([]label.LabeledResource, len(set))
	for i := range set {
		ll[i] = set[i]
	}

	return ll
}
