<template>
  <v-dialog
    v-model="dialogVisible"
    max-width="500"
    persistent
  >
    <v-card :title="isEdit ? $t('apiKey.edit') : $t('apiKey.create')" rounded="lg">
      <v-divider />
      <v-card-text>
        <v-text-field
          v-model="form.name"
          :label="$t('apiKey.name')"
          :placeholder="$t('apiKey.namePlaceholder')"
          variant="outlined"
          density="compact"
          hide-details
          class="mb-4"
        />
        <v-switch
          v-if="isEdit"
          v-model="form.enable"
          :label="$t('enable')"
          color="primary"
          hide-details
        />
        <div v-if="isEdit && form.key" class="mt-4">
          <v-label class="mb-2">{{ $t('apiKey.key') }}</v-label>
          <div class="d-flex align-center">
            <code class="text-primary flex-grow-1">{{ form.key }}</code>
            <v-btn size="small" variant="text" icon @click="copyKey">
              <v-icon icon="mdi-content-copy" />
            </v-btn>
          </div>
        </div>
      </v-card-text>
      <v-divider />
      <v-card-actions>
        <v-spacer />
        <v-btn color="grey" variant="outlined" @click="close">
          {{ $t('close') }}
        </v-btn>
        <v-btn color="primary" variant="flat" @click="save" :disabled="!form.name">
          {{ $t('actions.save') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts" setup>
import Data from '@/store/modules/data'
import { ref, watch, computed } from 'vue'
import { i18n } from '@/locales'
import { push } from 'notivue'

const props = defineProps<{
  visible: boolean
  apiKeyId: number
}>()

const emit = defineEmits(['close', 'update:modelValue'])

const form = ref({
  name: '',
  enable: true,
  key: ''
})

const isEdit = computed(() => props.apiKeyId > 0)

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => {
    emit('update:modelValue', val)
    if (!val) emit('close')
  }
})

watch(() => props.visible, async (newVal) => {
  if (newVal) {
    if (props.apiKeyId > 0) {
      const apiKey = Data().apiKeys.find(k => k.id === props.apiKeyId)
      if (apiKey) {
        form.value = {
          name: apiKey.name,
          enable: apiKey.enable,
          key: apiKey.key
        }
      }
    } else {
      form.value = { name: '', enable: true, key: '' }
    }
  }
})

const close = () => {
  emit('update:modelValue', false)
  emit('close')
}

const save = async () => {
  if (!form.value.name) return

  if (isEdit.value) {
    const success = await Data().updateApiKey(props.apiKeyId, form.value.name, form.value.enable)
    if (success) close()
  } else {
    const result = await Data().createApiKey(form.value.name)
    if (result) {
      push.success({
        title: i18n.global.t('apiKey.created'),
        message: i18n.global.t('apiKey.createdDesc')
      })
      close()
    }
  }
}

const copyKey = () => {
  navigator.clipboard.writeText(form.value.key)
  push.success({
    message: i18n.global.t('copied')
  })
}
</script>
