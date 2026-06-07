<template>
  <div v-if="items != null">
    <EditDialog
      v-model="editDialog"
      :save-button-text="itemId === 'new' ? $t('create') : $t('save')"
      :title="itemId === 'new' ? $t('newSlackNotification') : $t('editSlackNotification')"
      :max-width="450"
      :transition="false"
      @save="loadItems()"
    >
      <template v-slot:form="{ onSave, onError, needSave, needReset }">
        <SlackNotificationForm
          :project-id="projectId"
          :item-id="itemId"
          @save="onSave"
          @error="onError"
          :need-save="needSave"
          :need-reset="needReset"
        />
      </template>
    </EditDialog>

    <YesNoDialog
      :title="$t('deleteSlackNotification')"
      :text="$t('deleteSlackNotificationMsg')"
      v-model="deleteItemDialog"
      @yes="deleteItem(itemId)"
    />

    <v-toolbar flat>
      <v-app-bar-nav-icon @click="showDrawer()"></v-app-bar-nav-icon>
      <v-toolbar-title>{{ $t('slackNotifications') }}</v-toolbar-title>
      <v-spacer></v-spacer>
      <v-btn
        color="primary"
        v-if="can(USER_PERMISSIONS.manageProjectResources)"
        @click="editItem('new')"
      >{{ $t('newSlackNotification') }}
      </v-btn>
    </v-toolbar>

    <v-data-table
      :headers="headers"
      :items="items"
      class="mt-4"
      :items-per-page="Number.MAX_VALUE"
    >
      <template v-slot:item.color="{ item }">
        <span
          v-if="item.color"
          :style="{ backgroundColor: item.color, padding: '2px 12px', borderRadius: '3px' }"
        >&nbsp;</span>
        <span v-else>—</span>
      </template>
      <template v-slot:item.actions="{ item }">
        <v-btn-toggle dense :value-comparator="() => false">
          <v-btn @click="testNotification(item.id)" :loading="testingId === item.id">
            <v-icon>mdi-bell-ring-outline</v-icon>
          </v-btn>
          <v-btn @click="askDeleteItem(item.id)">
            <v-icon>mdi-delete</v-icon>
          </v-btn>
          <v-btn @click="editItem(item.id)">
            <v-icon>mdi-pencil</v-icon>
          </v-btn>
        </v-btn-toggle>
      </template>
    </v-data-table>
  </div>
</template>
<script>
import axios from 'axios';
import EventBus from '@/event-bus';
import { USER_PERMISSIONS } from '@/lib/constants';
import ItemListPageBase from '@/components/ItemListPageBase';
import SlackNotificationForm from '@/components/SlackNotificationForm.vue';

export default {
  mixins: [ItemListPageBase],
  components: { SlackNotificationForm },
  data() {
    return {
      testingId: null,
    };
  },
  methods: {
    async testNotification(id) {
      this.testingId = id;
      try {
        await axios({
          method: 'post',
          url: `/api/project/${this.projectId}/slack_notifications/${id}/test`,
        });
        EventBus.$emit('i-snackbar', { color: 'success', text: 'Test notification sent' });
      } catch (e) {
        EventBus.$emit('i-snackbar', { color: 'error', text: 'Failed to send test notification' });
      } finally {
        this.testingId = null;
      }
    },
    allowActions() {
      return this.can(USER_PERMISSIONS.updateProject);
    },
    getHeaders() {
      return [
        { text: this.$i18n.t('name'), value: 'name', sortable: true },
        { text: this.$i18n.t('slackChannel'), value: 'channel', sortable: true },
        { text: this.$i18n.t('slackColor'), value: 'color', sortable: false },
        { value: 'actions', sortable: false, width: '0%' },
      ];
    },
    getItemsUrl() {
      return `/api/project/${this.projectId}/slack_notifications`;
    },
    getSingleItemUrl() {
      return `/api/project/${this.projectId}/slack_notifications/${this.itemId}`;
    },
    getEventName() {
      return 'w-slack-notification';
    },
  },
};
</script>
