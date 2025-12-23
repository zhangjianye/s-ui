import HttpUtils from '@/plugins/httputil'
import { defineStore } from 'pinia'
import { push } from 'notivue'
import { i18n } from '@/locales'
import { Inbound } from '@/types/inbounds'
import { Client } from '@/types/clients'
import { Node, NodeToken, NodeOnlines } from '@/types/node'
import { ApiKey, WebhookConfig } from '@/types/apikey'

const Data = defineStore('Data', {
  state: () => ({
    lastLoad: 0,
    reloadItems: localStorage.getItem("reloadItems")?.split(',')?? <string[]>[],
    subURI: "",
    enableTraffic: false,
    onlines: {inbound: <string[]>[], outbound: <string[]>[], user: <string[]>[]},
    config: <any>{},
    inbounds: <any[]>[],
    outbounds: <any[]>[],
    services: <any[]>[],
    endpoints: <any[]>[],
    clients: <any>[],
    tlsConfigs: <any[]>[],
    // 节点管理 (UAP)
    nodeMode: 'standalone' as 'standalone' | 'master' | 'worker',
    isReadOnly: false,
    nodes: <Node[]>[],
    nodeTokens: <NodeToken[]>[],
    nodeOnlines: <NodeOnlines[]>[],
    // API Key 管理
    apiKeys: <ApiKey[]>[],
    webhookConfig: <WebhookConfig>{ callbackUrl: '', callbackSecret: '', enable: false },
  }),
  actions: {
    async loadData() {
      const msg = await HttpUtils.get('api/load', this.lastLoad >0 ? {lu: this.lastLoad} : {} )
      if(msg.success) {
        this.onlines = msg.obj.onlines
        if (msg.obj.lastLog) {
          push.error({
            title: i18n.global.t('error.core'),
            duration: 5000,
            message: msg.obj.lastLog
          })
        }
        
        if (msg.obj.config) {
          this.setNewData(msg.obj)
        }
      }
    },
    setNewData(data: any) {
      this.lastLoad = Math.floor((new Date()).getTime()/1000)
      if (data.subURI) this.subURI = data.subURI
      if (data.enableTraffic) this.enableTraffic = data.enableTraffic
      if (data.config) this.config = data.config
      if (Object.hasOwn(data, 'clients')) this.clients = data.clients ?? []
      if (Object.hasOwn(data, 'inbounds')) this.inbounds = data.inbounds ?? []
      if (Object.hasOwn(data, 'outbounds')) this.outbounds = data.outbounds ?? []
      if (Object.hasOwn(data, 'services')) this.services = data.services ?? []
      if (Object.hasOwn(data, 'endpoints')) this.endpoints = data.endpoints ?? []
      if (Object.hasOwn(data, 'tls')) this.tlsConfigs = data.tls ?? []
    },
    async loadInbounds(ids: number[]): Promise<Inbound[]> {
      const options = ids.length > 0 ? {id: ids.join(",")} : {}
      const msg = await HttpUtils.get('api/inbounds', options)
      if(msg.success) {
        return msg.obj.inbounds
      }
      return <Inbound[]>[]
    },
    async loadClients(id: number): Promise<Client> {
      const options = id > 0 ? {id: id} : {}
      const msg = await HttpUtils.get('api/clients', options)
      if(msg.success) {
        return <Client>msg.obj.clients[0]??{}
      }
      return <Client>{}
    },
    async save (object: string, action: string, data: any, initUsers?: number[]): Promise<boolean> {
      let postData = {
        object: object,
        action: action,
        data: JSON.stringify(data, null, 2),
        initUsers: initUsers?.join(',') ?? undefined
      }
      const msg = await HttpUtils.post('api/save', postData)
      if (msg.success) {
        const objectName = ['tls', 'config'].includes(object) ? object : object.substring(0, object.length - 1)
        push.success({
          title: i18n.global.t('success'),
          duration: 5000,
          message: i18n.global.t('actions.' + action) + " " + i18n.global.t('objects.' + objectName)
        })
        this.setNewData(msg.obj)
      }
      return msg.success
    },
    // Check duplicate client name
    checkClientName (id: number, newName: string): boolean {
      const oldName = id > 0 ? this.clients.findLast((i: any) => i.id == id)?.name : null
      if (newName != oldName && this.clients.findIndex((c: any) => c.name == newName) != -1) {
        push.error({
          message: i18n.global.t('error.dplData') + ": " + i18n.global.t('client.name')
        })
        return true
      }
      return false
    },
    // Check bulk client names
    checkBulkClientNames (names: string[]): boolean {
      const newNames = new Set(names)
      const oldNames = new Set(this.clients.map((c: any) => c.name))
      const allNames = new Set([...oldNames, ...newNames])
      console.log(oldNames, newNames, allNames)
      if (newNames.size != names.length || oldNames.size + newNames.size != allNames.size) {
        push.error({
          message: i18n.global.t('error.dplData') + ": " + i18n.global.t('client.name')
        })
        return true
      }
      return false
    },
    // check duplicate tag
    checkTag (object: string, id: number, tag: string): boolean {
      let objects = <any[]>[]
      switch (object) {
        case 'inbound':
          objects = this.inbounds
          break
        case 'outbound':
          objects = this.outbounds
          break
        case 'service':
          objects = this.services
          break
        case 'endpoint':
          objects = this.endpoints
          break
        default:
          return false
      }
      const oldObject = id > 0 ? objects.findLast((i: any) => i.id == id) : null
      if (tag != oldObject?.tag && objects.findIndex((i: any) => i.tag == tag) != -1) {
        push.error({
          message: i18n.global.t('error.dplData') + ": " + i18n.global.t('objects.tag')
        })
        return true
      }
      return false
    },
    // ========== 节点管理 API (UAP) ==========
    async loadNodeMode(): Promise<void> {
      const msg = await HttpUtils.get('api/nodeMode')
      if (msg.success) {
        this.nodeMode = msg.obj.mode ?? 'standalone'
        this.isReadOnly = msg.obj.isReadOnly ?? false
      }
    },
    async loadNodes(): Promise<void> {
      const msg = await HttpUtils.get('api/nodes')
      if (msg.success) {
        this.nodes = msg.obj ?? []
      }
    },
    async loadNodeTokens(): Promise<void> {
      const msg = await HttpUtils.get('api/nodeTokens')
      if (msg.success) {
        this.nodeTokens = msg.obj ?? []
      }
    },
    async loadNodeOnlines(): Promise<void> {
      // TODO: implement when endpoint is ready
      this.nodeOnlines = []
    },
    async generateNodeToken(name: string, expiresAt: number): Promise<NodeToken | null> {
      const msg = await HttpUtils.post('api/generateNodeToken', { name, expiresAt })
      if (msg.success) {
        push.success({
          title: i18n.global.t('success'),
          message: i18n.global.t('node.tokenGenerated')
        })
        await this.loadNodeTokens()
        return msg.obj
      }
      return null
    },
    async deleteNodeToken(id: number): Promise<boolean> {
      const msg = await HttpUtils.post('api/deleteNodeToken', { id })
      if (msg.success) {
        push.success({
          title: i18n.global.t('success'),
          message: i18n.global.t('actions.del') + ' ' + i18n.global.t('node.token')
        })
        await this.loadNodeTokens()
        return true
      }
      return false
    },
    async updateNode(node: Node): Promise<boolean> {
      const msg = await HttpUtils.post('api/save', {
        object: 'nodes',
        action: 'edit',
        data: JSON.stringify(node)
      })
      if (msg.success) {
        push.success({
          title: i18n.global.t('success'),
          message: i18n.global.t('actions.save') + ' ' + i18n.global.t('objects.node')
        })
        await this.loadNodes()
        return true
      }
      return false
    },
    async deleteNode(id: number): Promise<boolean> {
      const msg = await HttpUtils.post('api/save', {
        object: 'nodes',
        action: 'del',
        data: JSON.stringify({ id })
      })
      if (msg.success) {
        push.success({
          title: i18n.global.t('success'),
          message: i18n.global.t('actions.del') + ' ' + i18n.global.t('objects.node')
        })
        await this.loadNodes()
        return true
      }
      return false
    },
    async syncNode(id: number): Promise<boolean> {
      // TODO: implement sync endpoint
      push.info({ message: 'Sync not implemented yet' })
      return false
    },
    // ========== API Key 管理 ==========
    async loadApiKeys(): Promise<void> {
      const msg = await HttpUtils.get('api/apiKeys')
      if (msg.success) {
        this.apiKeys = msg.obj ?? []
      }
    },
    async createApiKey(name: string): Promise<ApiKey | null> {
      const msg = await HttpUtils.post('api/createApiKey', { name })
      if (msg.success) {
        push.success({
          title: i18n.global.t('success'),
          message: i18n.global.t('actions.new') + ' ' + i18n.global.t('apiKey.title')
        })
        await this.loadApiKeys()
        return msg.obj
      }
      return null
    },
    async updateApiKey(id: number, name: string, enable: boolean): Promise<boolean> {
      const msg = await HttpUtils.post('api/updateApiKey', { id, name, enable: enable ? 'true' : 'false' })
      if (msg.success) {
        push.success({
          title: i18n.global.t('success'),
          message: i18n.global.t('actions.save') + ' ' + i18n.global.t('apiKey.title')
        })
        await this.loadApiKeys()
        return true
      }
      return false
    },
    async deleteApiKey(id: number): Promise<boolean> {
      const msg = await HttpUtils.post('api/deleteApiKey', { id })
      if (msg.success) {
        push.success({
          title: i18n.global.t('success'),
          message: i18n.global.t('actions.del') + ' ' + i18n.global.t('apiKey.title')
        })
        await this.loadApiKeys()
        return true
      }
      return false
    },
    // ========== Webhook 配置 ==========
    async loadWebhookConfig(): Promise<void> {
      const msg = await HttpUtils.get('api/webhookConfig')
      if (msg.success) {
        this.webhookConfig = msg.obj ?? { callbackUrl: '', callbackSecret: '', enable: false }
      }
    },
    async saveWebhookConfig(config: WebhookConfig): Promise<boolean> {
      const msg = await HttpUtils.post('api/saveWebhookConfig', {
        callbackUrl: config.callbackUrl,
        callbackSecret: config.callbackSecret,
        enable: config.enable ? 'true' : 'false'
      })
      if (msg.success) {
        push.success({
          title: i18n.global.t('success'),
          message: i18n.global.t('actions.save') + ' ' + i18n.global.t('webhook.title')
        })
        await this.loadWebhookConfig()
        return true
      }
      return false
    },
  }
})

export default Data