package compose

import (
	"github.com/cortezaproject/corteza/server/codegen/schema"
)

module: {
	handle: "module"
	parents: [
		{handle: "namespace"},
	]

	model: {
		ident: "compose_module"
		attributes: {
			id: schema.IdField
			namespace_id: {
				ident: "namespaceID",
				goType: "uint64",
				storeIdent: "rel_namespace"
				dal: { type: "Ref", refModelResType: "corteza::compose:namespace" }
			}
			handle: schema.HandleField
			name: {
				sortable: true
				dal: {}
			}
			meta: {
				goType: "rawJson"
				dal: { type: "JSON", defaultEmptyObject: true }
			}
			config: {
				goType: "types.ModuleConfig"
				dal: { type: "JSON", defaultEmptyObject: true }
			}
			fields: {
				goType: "types.ModuleFieldSet",
				store: false
			}
			created_at: schema.SortableTimestampNowField
			updated_at: schema.SortableTimestampNilField
			deleted_at: schema.SortableTimestampNilField
		}

		indexes: {
			"primary": { attribute: "id" }
			"namespace": { attribute: "namespace_id" },
			"unique_handle": {
				fields: [{ attribute: "handle", modifiers: ["LOWERCASE"] }, { attribute: "namespace_id" }]
				predicate: "handle != '' AND deleted_at IS NULL"
			}
		}
	}

	filter: {
		struct: {
			module_id: { goType: "[]uint64", ident: "moduleID", storeIdent: "id" }
			namespace_id: { goType: "uint64", ident: "namespaceID", storeIdent: "rel_namespace" }
			handle: { goType: "string" }
			name: { goType: "string" }
			deleted: { goType: "filter.State", storeIdent: "deleted_at" }
		}

		query: ["handle", "name"]
		byValue: ["handle", "module_id", "namespace_id"]
		byNilState: ["deleted"]
	}

	rbac: {
		operations: {
			"read": {}
			"update": {}
			"delete": {}
			"record.create": description:  "Create record"
			"owned-record.create": description:  "Create record with custom owner"
			"records.search": description: "List, search or filter records"
		}
	}

	store: {
		ident: "composeModule"

		api: {
			lookups: [
				{
					fields: ["namespace_id", "handle"]
					constraintCheck: true
					nullConstraint: ["deleted_at"]
					description: """
						searches for compose module by handle (case-insensitive)
						"""
				}, {
					fields: ["namespace_id", "name"]
					nullConstraint: ["deleted_at"]
					description: """
						searches for compose module by name (case-insensitive)
						"""
				}, {
					fields: ["id"]
					description: """
						searches for compose module by ID

						It returns compose module even if deleted
						"""
				},
			]
		}
	}

	locale: {
		extended: true

		keys: {
			"name": {}
		}
	}

	//locale:
	//  resource:
	//    references: [ namespace, ID ]
	//
	//  extended: true
	//  keys:
	//    - name
}
