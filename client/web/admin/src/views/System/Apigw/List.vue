<template>
  <b-container
    class="py-3"
  >
    <c-content-header
      :title="$t('title')"
    >
      <span
        class="text-nowrap"
      >
        <b-button
          v-if="canCreate"
          data-test-id="button-add"
          variant="primary"
          :to="{ name: 'system.apigw.new' }"
        >
          {{ $t('new') }}
        </b-button>
        <b-button
          v-if="$Settings.get('apigw.profilerEnabled', false)"
          data-test-id="button-profiler"
          class="ml-2"
          variant="info"
          :to="{ name: 'system.apigw.profiler' }"
        >
          {{ $t('profiler') }}
        </b-button>
        <c-permissions-button
          v-if="canGrant"
          data-test-id="button-permissions"
          resource="corteza::system:apigw-route/*"
          button-variant="light"
          class="ml-2"
        >
          <font-awesome-icon :icon="['fas', 'lock']" />
          {{ $t('permissions') }}
        </c-permissions-button>
      </span>
      <b-dropdown
        v-if="false"
        variant="link"
        right
        menu-class="shadow-sm"
        :text="$t('export')"
      >
        <b-dropdown-item-button variant="link">
          {{ $t('yaml') }}
        </b-dropdown-item-button>
      </b-dropdown>
    </c-content-header>
    <c-resource-list
      :primary-key="primaryKey"
      :filter="filter"
      :sorting="sorting"
      :pagination="pagination"
      :fields="fields"
      :items="items"
      :row-class="genericRowClass"
      :translations="{
        searchPlaceholder: $t('filterForm.query.placeholder'),
        notFound: $t('admin:general.notFound'),
        noItems: $t('admin:general.resource-list.no-items'),
        loading: $t('loading'),
        showingPagination: 'admin:general.pagination.showing',
        singlePluralPagination: 'admin:general.pagination.single',
        prevPagination: $t('admin:general.pagination.prev'),
        nextPagination: $t('admin:general.pagination.next'),
      }"
      @search="filterList"
    >
      <template #header>
        <c-resource-list-status-filter
          v-model="filter.deleted"
          data-test-id="filter-deleted-routes"
          :label="$t('filterForm.deleted.label')"
          :excluded-label="$t('filterForm.excluded.label')"
          :inclusive-label="$t('filterForm.inclusive.label')"
          :exclusive-label="$t('filterForm.exclusive.label')"
          @change="filterList"
        />
      </template>

      <template #actions="{ item }">
        <b-button
          size="sm"
          variant="link"
          :to="{ name: editRoute, params: { [primaryKey]: item[primaryKey] } }"
        >
          <font-awesome-icon
            :icon="['fas', 'pen']"
          />
        </b-button>
      </template>
    </c-resource-list>
  </b-container>
</template>

<script>
import * as moment from 'moment'
import listHelpers from 'corteza-webapp-admin/src/mixins/listHelpers'
import { mapGetters } from 'vuex'
import { components } from '@cortezaproject/corteza-vue'
const { CResourceList } = components

export default {
  components: {
    CResourceList,
  },

  mixins: [
    listHelpers,
  ],

  i18nOptions: {
    namespaces: [ 'system.apigw' ],
    keyPrefix: 'list',
  },

  data () {
    return {
      id: 'routes',

      primaryKey: 'routeID',
      editRoute: 'system.apigw.edit',

      filter: {
        query: '',
        deleted: 0,
      },

      sorting: {
        sortBy: 'createdAt',
        sortDesc: true,
      },

      fields: [
        {
          key: 'endpoint',
          sortable: true,
        },
        {
          key: 'method',
          sortable: false,
        },
        {
          key: 'enabled',
          formatter: (v) => v ? 'Yes' : 'No',
        },
        {
          key: 'createdAt',
          sortable: true,
          formatter: (v) => moment(v).fromNow(),
        },
        {
          key: 'actions',
          label: '',
          tdClass: 'text-right',
        },
      ].map(c => ({
        ...c,
        // Generate column label translation key
        label: this.$t(`columns.${c.key}`),
      })),
    }
  },

  computed: {
    ...mapGetters({
      can: 'rbac/can',
    }),

    canCreate () {
      return this.can('system/', 'apigw-route.create')
    },

    canGrant () {
      return this.can('system/', 'grant')
    },
  },

  methods: {
    items () {
      return this.procListResults(this.$SystemAPI.apigwRouteList(this.encodeListParams()))
    },
  },
}
</script>
