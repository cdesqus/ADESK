export interface WhatsAppSession {
  id: string;
  name: string;
  phoneNumber: string;
  status: 'active' | 'pending' | 'disconnected';
  createdAt: string; // ISO timestamp
  qrCode?: string; // base64
}

export interface EngineerWAPhone {
  id: string;
  engineerId: string;
  engineerName: string;
  phoneNumber: string;
  groupId?: string;
  isActive: boolean;
  createdAt: string; // ISO timestamp
}

export interface WAHookStatus {
  status: 'connected' | 'disconnected';
  lastMessageAt?: string; // ISO timestamp
  errorLog?: string[];
}
