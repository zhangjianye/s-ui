export interface Node {
  id: number
  name: string
  address: string
  externalHost?: string
  externalPort?: number
  flag?: string
  country?: string
  city?: string
  isPremium?: boolean
  enable: boolean
  status: 'online' | 'offline' | 'error'
  version?: string
  lastSeen?: number
}

export interface NodeToken {
  id: number
  token: string
  name: string
  expiresAt: number
  used: boolean
  usedBy?: string
}

export interface NodeOnlines {
  nodeId: number
  nodeName: string
  inbound: string[]
  outbound: string[]
  user: string[]
}

export const getStatusColor = (status: string): string => {
  switch (status) {
    case 'online': return 'success'
    case 'offline': return 'grey'
    case 'error': return 'error'
    default: return 'grey'
  }
}

export const getStatusIcon = (status: string): string => {
  switch (status) {
    case 'online': return 'mdi-check-circle'
    case 'offline': return 'mdi-circle-outline'
    case 'error': return 'mdi-alert-circle'
    default: return 'mdi-help-circle'
  }
}
