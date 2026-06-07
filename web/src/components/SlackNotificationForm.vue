<template>
  <v-form ref="form" lazy-validation v-model="formValid" v-if="item != null">
    <v-alert :value="formError" color="error" class="pb-2">{{ formError }}</v-alert>

    <v-text-field
      v-model="item.name"
      :label="$t('name')"
      :rules="[(v) => !!v || $t('name_required')]"
      required
      :disabled="formSaving"
      outlined
      dense
    />

    <v-text-field
      v-model="item.description"
      :label="$t('description')"
      :disabled="formSaving"
      outlined
      dense
    />

    <v-divider class="mb-4" />

    <v-text-field
      v-model="item.token"
      label="Token"
      :rules="[(v) => !!v || $t('required')]"
      required
      :disabled="formSaving"
      type="password"
      outlined
      dense
    />

    <v-text-field
      v-model="item.channel"
      label="Destination Channels"
      :rules="[(v) => !!v || $t('required')]"
      required
      :disabled="formSaving"
      hint="#channel1, #channel2"
      outlined
      dense
    />

    <v-text-field
      v-model="item.color"
      label="Notification color"
      :disabled="formSaving"
      hint="#3af or #789abc"
      outlined
      dense
    />

    <v-divider class="mb-4" />

    <v-checkbox
      v-model="showCustomMessages"
      label="Customize messages"
      class="mt-0"
    />

    <div v-if="showCustomMessages">
      <v-text-field
        v-model="item.started_message"
        label="Start message"
        :disabled="formSaving"
        outlined
        dense
      />

      <v-text-field
        v-model="item.success_message"
        label="Success message"
        :disabled="formSaving"
        outlined
        dense
      />

      <v-text-field
        v-model="item.error_message"
        label="Error message"
        :disabled="formSaving"
        outlined
        dense
      />
    </div>
  </v-form>
</template>
<script>
import ItemFormBase from '@/components/ItemFormBase';

export default {
  mixins: [ItemFormBase],
  data() {
    return {
      showCustomMessages: false,
    };
  },
  watch: {
    item(val) {
      if (val && (val.started_message || val.success_message || val.error_message)) {
        this.showCustomMessages = true;
      }
    },
  },
  methods: {
    getItemsUrl() {
      return `/api/project/${this.projectId}/slack_notifications`;
    },
    getSingleItemUrl() {
      return `/api/project/${this.projectId}/slack_notifications/${this.itemId}`;
    },
  },
};
</script>
