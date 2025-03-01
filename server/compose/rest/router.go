package rest

import (
	"github.com/go-chi/chi/v5"

	"github.com/cortezaproject/corteza/server/compose/rest/handlers"
	"github.com/cortezaproject/corteza/server/pkg/auth"
)

func MountRoutes() func(r chi.Router) {
	return func(r chi.Router) {
		var (
			namespace    = Namespace{}.New()
			module       = Module{}.New()
			record       = Record{}.New()
			page         = Page{}.New()
			chart        = Chart{}.New()
			notification = Notification{}.New()
			attachment   = Attachment{}.New()
			automation   = Automation{}.New()
			dataPrivacy  = DataPrivacy{}.New()
		)

		// Initialize handlers & controllers.
		r.Group(func(r chi.Router) {
			// Use alternative handlers that support file serving
			handlers.NewAttachment(attachment).MountRoutes(r)
		})

		// Protect all _private_ routes
		r.Group(func(r chi.Router) {
			r.Use(auth.HttpTokenValidator("api"))

			handlers.NewPermissions(Permissions{}.New()).MountRoutes(r)
			handlers.NewNamespace(namespace).MountRoutes(r)
			handlers.NewPage(page).MountRoutes(r)
			handlers.NewAutomation(automation).MountRoutes(r)
			handlers.NewModule(module).MountRoutes(r)
			handlers.NewRecord(record).MountRoutes(r)
			handlers.NewChart(chart).MountRoutes(r)
			handlers.NewNotification(notification).MountRoutes(r)
			handlers.NewDataPrivacy(dataPrivacy).MountRoutes(r)
		})
	}
}
