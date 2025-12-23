<template>
  <NodeModal
    v-model="nodeModal.visible"
    :visible="nodeModal.visible"
    :node="nodeModal.node"
    @close="closeNodeModal"
    @save="saveNode"
  />
  <TokenModal
    v-model="tokenModal.visible"
    :visible="tokenModal.visible"
    @close="closeTokenModal"
    @generate="generateToken"
  />
  <v-card :loading="loading">
    <v-tabs v-model="tab" color="primary" align-tabs="center">
      <v-tab value="nodes">{{ $t('node.nodes') }}</v-tab>
      <v-tab value="tokens">{{ $t('node.tokens') }}</v-tab>
    </v-tabs>
    <v-card-text>
      <v-window v-model="tab">
        <!-- Nodes Tab -->
        <v-window-item value="nodes">
          <v-row justify="center" align="center" class="mb-4">
            <v-col cols="auto">
              <v-chip :color="modeColor" label>
                <v-icon start :icon="modeIcon"></v-icon>
                {{ $t('node.mode') }}: {{ $t('node.' + nodeMode) }}
              </v-chip>
            </v-col>
          </v-row>
          <v-data-table
            :headers="nodeHeaders"
            :items="nodes"
            :hide-default-footer="nodes.length <= 10"
            hide-no-data
            fixed-header
            item-value="id"
            class="elevation-3 rounded"
          >
            <template v-slot:item.status="{ item }">
              <v-chip
                :color="getStatusColor(item.status)"
                size="small"
                label
              >
                <v-icon start :icon="getStatusIcon(item.status)" size="small"></v-icon>
                {{ $t('node.' + item.status) }}
              </v-chip>
            </template>
            <template v-slot:item.enable="{ item }">
              <v-switch
                v-model="item.enable"
                color="primary"
                hide-details
                density="compact"
                @change="toggleNode(item)"
              ></v-switch>
            </template>
            <template v-slot:item.isPremium="{ item }">
              <v-icon v-if="item.isPremium" color="warning" icon="mdi-star"></v-icon>
              <span v-else>-</span>
            </template>
            <template v-slot:item.lastSeen="{ item }">
              {{ item.lastSeen ? formatTime(item.lastSeen) : '-' }}
            </template>
            <template v-slot:item.actions="{ item }">
              <v-icon class="me-2" @click="editNode(item)">mdi-pencil</v-icon>
              <v-icon class="me-2" color="primary" @click="syncNodeData(item.id)">mdi-sync</v-icon>
              <v-menu
                v-model="delOverlay[nodes.findIndex(n => n.id == item.id)]"
                :close-on-content-click="false"
                location="top center"
              >
                <template v-slot:activator="{ props }">
                  <v-icon color="error" v-bind="props">mdi-delete</v-icon>
                </template>
                <v-card :title="$t('actions.del')" rounded="lg">
                  <v-divider></v-divider>
                  <v-card-text>{{ $t('confirm') }}</v-card-text>
                  <v-card-actions>
                    <v-btn color="error" variant="outlined" @click="delNode(item.id)">{{ $t('yes') }}</v-btn>
                    <v-btn color="success" variant="outlined" @click="delOverlay[nodes.findIndex(n => n.id == item.id)] = false">{{ $t('no') }}</v-btn>
                  </v-card-actions>
                </v-card>
              </v-menu>
            </template>
          </v-data-table>
        </v-window-item>

        <!-- Tokens Tab -->
        <v-window-item value="tokens">
          <v-row justify="center" align="center" class="mb-4">
            <v-col cols="auto">
              <v-btn color="primary" @click="showTokenModal">{{ $t('node.generateToken') }}</v-btn>
            </v-col>
          </v-row>
          <v-data-table
            :headers="tokenHeaders"
            :items="nodeTokens"
            :hide-default-footer="nodeTokens.length <= 10"
            hide-no-data
            fixed-header
            item-value="id"
            class="elevation-3 rounded"
          >
            <template v-slot:item.token="{ item }">
              <code class="text-caption">{{ item.token.substring(0, 20) }}...</code>
              <v-btn
                size="x-small"
                variant="text"
                icon="mdi-content-copy"
                @click="copyToken(item.token)"
              ></v-btn>
            </template>
            <template v-slot:item.expiresAt="{ item }">
              <v-chip
                :color="item.expiresAt < Date.now() / 1000 ? 'error' : 'success'"
                size="small"
                label
              >
                {{ formatTime(item.expiresAt) }}
              </v-chip>
            </template>
            <template v-slot:item.used="{ item }">
              <v-chip
                :color="item.used ? 'success' : 'grey'"
                size="small"
                label
              >
                {{ item.used ? $t('yes') : $t('no') }}
              </v-chip>
            </template>
            <template v-slot:item.actions="{ item }">
              <v-menu
                v-model="tokenDelOverlay[nodeTokens.findIndex(t => t.id == item.id)]"
                :close-on-content-click="false"
                location="top center"
              >
                <template v-slot:activator="{ props }">
                  <v-icon color="error" v-bind="props">mdi-delete</v-icon>
                </template>
                <v-card :title="$t('actions.del')" rounded="lg">
                  <v-divider></v-divider>
                  <v-card-text>{{ $t('confirm') }}</v-card-text>
                  <v-card-actions>
                    <v-btn color="error" variant="outlined" @click="delToken(item.id)">{{ $t('yes') }}</v-btn>
                    <v-btn color="success" variant="outlined" @click="tokenDelOverlay[nodeTokens.findIndex(t => t.id == item.id)] = false">{{ $t('no') }}</v-btn>
                  </v-card-actions>
                </v-card>
              </v-menu>
            </template>
          </v-data-table>
        </v-window-item>
      </v-window>
    </v-card-text>
  </v-card>
</template>

<script lang="ts" setup>
import { computed, inject, onMounted, ref, Ref } from 'vue'
import Data from '@/store/modules/data'
import { Node, getStatusColor, getStatusIcon } from '@/types/node'
import { i18n } from '@/locales'
import { push } from 'notivue'
import NodeModal from '@/layouts/modals/Node.vue'
import TokenModal from '@/layouts/modals/NodeToken.vue'

const loading: Ref = inject('loading') ?? ref(false)
const tab = ref('nodes')

const nodeMode = computed(() => Data().nodeMode)
const nodes = computed(() => Data().nodes)
const nodeTokens = computed(() => Data().nodeTokens)

const modeColor = computed(() => {
  switch (nodeMode.value) {
    case 'master': return 'primary'
    case 'worker': return 'warning'
    default: return 'grey'
  }
})

const modeIcon = computed(() => {
  switch (nodeMode.value) {
    case 'master': return 'mdi-server'
    case 'worker': return 'mdi-server-network'
    default: return 'mdi-server-off'
  }
})

const nodeHeaders = [
  { title: i18n.global.t('node.name'), key: 'name' },
  { title: i18n.global.t('node.address'), key: 'address' },
  { title: i18n.global.t('node.country'), key: 'country' },
  { title: i18n.global.t('node.status'), key: 'status' },
  { title: i18n.global.t('node.enable'), key: 'enable', width: 80 },
  { title: i18n.global.t('node.premium'), key: 'isPremium', width: 80 },
  { title: i18n.global.t('node.lastSeen'), key: 'lastSeen' },
  { title: i18n.global.t('actions.action'), key: 'actions', sortable: false },
]

const tokenHeaders = [
  { title: i18n.global.t('node.tokenName'), key: 'name' },
  { title: i18n.global.t('node.token'), key: 'token' },
  { title: i18n.global.t('node.expiresAt'), key: 'expiresAt' },
  { title: i18n.global.t('node.used'), key: 'used' },
  { title: i18n.global.t('node.usedBy'), key: 'usedBy' },
  { title: i18n.global.t('actions.action'), key: 'actions', sortable: false },
]

const delOverlay = ref(new Array<boolean>(100).fill(false))
const tokenDelOverlay = ref(new Array<boolean>(100).fill(false))

const nodeModal = ref({
  visible: false,
  node: null as Node | null,
})

const tokenModal = ref({
  visible: false,
})

onMounted(async () => {
  loading.value = true
  await Promise.all([
    Data().loadNodeMode(),
    Data().loadNodes(),
    Data().loadNodeTokens(),
  ])
  loading.value = false
})

const formatTime = (timestamp: number) => {
  return new Date(timestamp * 1000).toLocaleString()
}

const editNode = (node: Node) => {
  nodeModal.value.node = { ...node }
  nodeModal.value.visible = true
}

const closeNodeModal = () => {
  nodeModal.value.visible = false
  nodeModal.value.node = null
}

const saveNode = async (node: Node) => {
  loading.value = true
  await Data().updateNode(node)
  closeNodeModal()
  loading.value = false
}

const toggleNode = async (node: Node) => {
  loading.value = true
  await Data().updateNode(node)
  loading.value = false
}

const delNode = async (id: number) => {
  loading.value = true
  const success = await Data().deleteNode(id)
  if (success) {
    const idx = nodes.value.findIndex(n => n.id === id)
    if (idx >= 0) delOverlay.value[idx] = false
  }
  loading.value = false
}

const syncNodeData = async (id: number) => {
  loading.value = true
  await Data().syncNode(id)
  loading.value = false
}

const showTokenModal = () => {
  tokenModal.value.visible = true
}

const closeTokenModal = () => {
  tokenModal.value.visible = false
}

const generateToken = async (name: string, expiresAt: number) => {
  loading.value = true
  const token = await Data().generateNodeToken(name, expiresAt)
  if (token) {
    closeTokenModal()
  }
  loading.value = false
}

const delToken = async (id: number) => {
  loading.value = true
  const success = await Data().deleteNodeToken(id)
  if (success) {
    const idx = nodeTokens.value.findIndex(t => t.id === id)
    if (idx >= 0) tokenDelOverlay.value[idx] = false
  }
  loading.value = false
}

const copyToken = async (token: string) => {
  try {
    await navigator.clipboard.writeText(token)
    push.success({
      message: i18n.global.t('actions.copied'),
      duration: 2000,
    })
  } catch (e) {
    push.error({
      message: 'Failed to copy',
      duration: 2000,
    })
  }
}
</script>
