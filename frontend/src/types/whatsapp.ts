export interface WhatsAppSession {
  id: string;
  session_name: string;
  phoneNumber: string;
  status: 'active' | 'pending' | 'disconnected' | 'UNKNOWN' | 'PENDING' | 'CONNECTED';
  createdAt: string; // ISO timestamp
  qr_code?: string; // base64
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
