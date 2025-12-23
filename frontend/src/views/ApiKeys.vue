<template>
  <ApiKeyModal
    v-model="modal.visible"
    :visible="modal.visible"
    :apiKeyId="modal.id"
    @close="closeModal"
  />

  <!-- 操作按钮 -->
  <v-row justify="center" align="center">
    <v-col cols="auto">
      <v-btn color="primary" @click="showModal(0)">
        <v-icon icon="mdi-key-plus" start />
        {{ $t('apiKey.create') }}
      </v-btn>
    </v-col>
    <v-col cols="auto">
      <v-btn variant="outlined" @click="refreshData">
        <v-icon icon="mdi-refresh" start />
        {{ $t('actions.update') }}
      </v-btn>
    </v-col>
  </v-row>

  <!-- API Key 列表 -->
  <v-row class="mt-4">
    <v-col cols="12">
      <v-card class="elevation-3 rounded">
        <v-card-title class="d-flex align-center">
          <v-icon icon="mdi-key-chain" class="me-2" />
          {{ $t('apiKey.list') }}
          <v-chip class="ms-2" size="small" color="primary">{{ apiKeys.length }}</v-chip>
        </v-card-title>
        <v-divider />
        <v-data-table
          :headers="headers"
          :items="apiKeys"
          :hide-default-footer="apiKeys.length <= 10"
          :items-per-page="10"
          item-value="id"
          :mobile="smAndDown"
          mobile-breakpoint="sm"
          class="rounded"
        >
          <template v-slot:item.key="{ item }">
            <code class="text-primary">{{ maskKey(item.key) }}</code>
            <v-btn size="x-small" variant="text" icon @click="copyKey(item.key)">
              <v-icon icon="mdi-content-copy" />
            </v-btn>
          </template>
          <template v-slot:item.enable="{ item }">
            <v-switch
              v-model="item.enable"
              color="primary"
              hide-details
              density="compact"
              @change="toggleKey(item)"
            />
          </template>
          <template v-slot:item.actions="{ item }">
            <v-icon
              class="me-2"
              @click="showModal(item.id)"
            >
              mdi-pencil
            </v-icon>
            <v-menu
              v-model="delOverlay[apiKeys.findIndex(k => k.id === item.id)]"
              :close-on-content-click="false"
              location="top center"
            >
              <template v-slot:activator="{ props }">
                <v-icon
                  color="error"
                  v-bind="props"
                >
                  mdi-delete
                </v-icon>
              </template>
              <v-card :title="$t('actions.del')" rounded="lg">
                <v-divider />
                <v-card-text>{{ $t('confirm') }}</v-card-text>
                <v-card-actions>
                  <v-btn color="error" variant="outlined" @click="deleteKey(item.id)">{{ $t('yes') }}</v-btn>
                  <v-btn color="success" variant="outlined" @click="delOverlay[apiKeys.findIndex(k => k.id === item.id)] = false">{{ $t('no') }}</v-btn>
                </v-card-actions>
              </v-card>
            </v-menu>
          </template>
        </v-data-table>
      </v-card>
    </v-col>
  </v-row>

  <!-- 使用说明 -->
  <v-row class="mt-4">
    <v-col cols="12">
      <v-card class="elevation-2 rounded">
        <v-card-title class="d-flex align-center">
          <v-icon icon="mdi-information" class="me-2" />
          {{ $t('apiKey.usage') }}
        </v-card-title>
        <v-divider />
        <v-card-text>
          <p class="mb-2">{{ $t('apiKey.usageDesc') }}</p>
          <code class="d-block pa-2 bg-grey-lighten-4 rounded">
            curl -H "X-API-Key: your-api-key" {{ baseUrl }}api/v1/users
          </code>
        </v-card-text>
      </v-card>
    </v-col>
  </v-row>
</template>

<script lang="ts" setup>
import Data from '@/store/modules/data'
import ApiKeyModal from '@/layouts/modals/ApiKey.vue'
import { computed, ref, onMounted } from 'vue'
import { i18n } from '@/locales'
import { useDisplay } from 'vuetify'
import { push } from 'notivue'

const { smAndDown } = useDisplay()

const apiKeys = computed(() => Data().apiKeys)

const headers = [
  { title: i18n.global.t('apiKey.name'), key: 'name' },
  { title: i18n.global.t('apiKey.key'), key: 'key' },
  { title: i18n.global.t('enable'), key: 'enable', width: 80 },
  { title: i18n.global.t('actions.action'), key: 'actions', sortable: false, width: 100 },
]

const modal = ref({ visible: false, id: 0 })
const delOverlay = ref<boolean[]>([])
const baseUrl = computed(() => window.location.origin + '/')

onMounted(async () => {
  await refreshData()
})

const refreshData = async () => {
  await Data().loadApiKeys()
  delOverlay.value = new Array(apiKeys.value.length).fill(false)
}

const showModal = (id: number) => {
  modal.value.id = id
  modal.value.visible = true
}

const closeModal = () => {
  modal.value.visible = false
}

const toggleKey = async (key: any) => {
  await Data().updateApiKey(key.id, key.name, key.enable)
}

const deleteKey = async (id: number) => {
  const index = apiKeys.value.findIndex(k => k.id === id)
  const success = await Data().deleteApiKey(id)
  if (success) delOverlay.value[index] = false
}

const copyKey = (key: string) => {
  navigator.clipboard.writeText(key)
  push.success({
    message: i18n.global.t('copied')
  })
}

const maskKey = (key: string) => {
  if (key.length <= 8) return key
  return key.substring(0, 4) + '****' + key.substring(key.length - 4)
}
</script>
