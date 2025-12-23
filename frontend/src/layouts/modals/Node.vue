<template>
  <v-dialog transition="dialog-bottom-transition" width="600">
    <v-card class="rounded-lg" :loading="loading">
      <v-card-title>
        <v-row>
          <v-col>{{ $t('objects.node') }}</v-col>
          <v-spacer></v-spacer>
          <v-col cols="auto"><v-icon icon="mdi-close-box" @click="$emit('close')" /></v-col>
        </v-row>
      </v-card-title>
      <v-divider></v-divider>
      <v-card-text v-if="localNode">
        <v-row>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="localNode.name"
              :label="$t('node.name')"
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="localNode.address"
              :label="$t('node.address')"
              hide-details
              disabled
            ></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="localNode.externalHost"
              :label="$t('node.externalHost')"
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model.number="localNode.externalPort"
              :label="$t('node.externalPort')"
              type="number"
              hide-details
            ></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="localNode.country"
              :label="$t('node.country')"
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="localNode.city"
              :label="$t('node.city')"
              hide-details
            ></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6">
            <v-text-field
              v-model="localNode.flag"
              :label="$t('node.flag')"
              hide-details
            ></v-text-field>
          </v-col>
          <v-col cols="12" sm="6">
            <v-switch
              v-model="localNode.isPremium"
              :label="$t('node.premium')"
              color="warning"
              hide-details
            ></v-switch>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12" sm="6">
            <v-switch
              v-model="localNode.enable"
              :label="$t('node.enable')"
              color="primary"
              hide-details
            ></v-switch>
          </v-col>
        </v-row>
      </v-card-text>
      <v-card-actions>
        <v-spacer></v-spacer>
        <v-btn
          color="blue-darken-1"
          variant="outlined"
          @click="$emit('close')"
        >
          {{ $t('actions.close') }}
        </v-btn>
        <v-btn
          color="blue-darken-1"
          variant="tonal"
          @click="save"
          :loading="loading"
        >
          {{ $t('actions.save') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts" setup>
import { ref, watch, computed } from 'vue'
import { Node } from '@/types/node'

const props = defineProps<{
  visible: boolean
  node: Node | null
}>()

const emit = defineEmits(['close', 'save'])

const loading = ref(false)
const localNode = ref<Node | null>(null)

const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => {
    if (!value) emit('close')
  }
})

watch(() => props.node, (newNode) => {
  if (newNode) {
    localNode.value = { ...newNode }
  } else {
    localNode.value = null
  }
}, { immediate: true })

const save = () => {
  if (localNode.value) {
    emit('save', localNode.value)
  }
}
</script>
