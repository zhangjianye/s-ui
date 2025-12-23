export interface ApiKey {
  id: number
  key: string
  name: string
  enable: boolean
  createdAt?: number
  lastUsedAt?: number
}

export interface WebhookConfig {
  id?: number
  callbackUrl: string
  callbackSecret: string
  enable: boolean
}
