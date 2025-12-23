<template>
  <v-dialog transition="dialog-bottom-transition" width="500">
    <v-card class="rounded-lg" :loading="loading">
      <v-card-title>
        <v-row>
          <v-col>{{ $t('node.generateToken') }}</v-col>
          <v-spacer></v-spacer>
          <v-col cols="auto"><v-icon icon="mdi-close-box" @click="$emit('close')" /></v-col>
        </v-row>
      </v-card-title>
      <v-divider></v-divider>
      <v-card-text>
        <v-row>
          <v-col cols="12">
            <v-text-field
              v-model="name"
              :label="$t('node.tokenName')"
              hide-details
            ></v-text-field>
          </v-col>
        </v-row>
        <v-row>
          <v-col cols="12">
            <v-text-field
              v-model.number="expiryDays"
              :label="$t('date.expiry')"
              type="number"
              :suffix="$t('date.d')"
              min="1"
              hide-details
            ></v-text-field>
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
          @click="generate"
          :loading="loading"
          :disabled="!name"
        >
          {{ $t('actions.add') }}
        </v-btn>
      </v-card-actions>
    </v-card>
  </v-dialog>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits(['close', 'generate'])

const loading = ref(false)
const name = ref('')
const expiryDays = ref(30)

watch(() => props.visible, (v) => {
  if (v) {
    name.value = ''
    expiryDays.value = 30
  }
})

const generate = () => {
  const expiresAt = Math.floor(Date.now() / 1000) + (expiryDays.value * 24 * 60 * 60)
  emit('generate', name.value, expiresAt)
}
</script>
