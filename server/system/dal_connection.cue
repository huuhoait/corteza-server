package system

import (
	"github.com/cortezaproject/corteza/server/codegen/schema"
)

dal_connection: {
	model: {
		attributes: {
			id:     schema.IdField
			handle: schema.HandleField
			type: {
				sortable: true
				dal: {}
			}

			config: {
				goType: "types.ConnectionConfig"
				dal: { type: "JSON", defaultEmptyObject: true }
			}
			meta: {
				goType: "types.ConnectionMeta"
				dal: { type: "JSON", defaultEmptyObject: true }
			}

			created_at: schema.SortableTimestampNowField
			updated_at: schema.SortableTimestampNilField
			deleted_at: schema.SortableTimestampNilField
			created_by: schema.AttributeUserRef
			updated_by: schema.AttributeUserRef
			deleted_by: schema.AttributeUserRef
		}

		indexes: {
			"primary": { attribute: "id" }
		}
	}

	filter: {
		struct: {
			connection_id: {goType: "[]uint64", ident: "connectionID", storeIdent: "id"}
			handle: {goType: "string"}
			type: {goType: "string"}

			deleted: {goType: "filter.State", storeIdent: "deleted_at"}
		}

		byValue: ["connection_id", "handle", "type"]
		byNilState: ["deleted"]
	}

	features: {
		labels: false
	}

	rbac: {
		operations: {
			"read": description:         "Read connection"
			"update": description:       "Update connection"
			"delete": description:       "Delete connection"
			"dal-config.manage": description: "Manage DAL configuration"
		}
	}

	store: {
		api: {
			lookups: [
				{
					fields: ["id"]
					description: """
						searches for connection by ID

						It returns connection even if deleted or suspended
						"""
				}, {
					fields: ["handle"]
					nullConstraint: ["deleted_at"]
					constraintCheck: true
					description: """
						searches for connection by handle

						It returns only valid connection (not deleted)
						"""
				},
			]
		}
	}
}
