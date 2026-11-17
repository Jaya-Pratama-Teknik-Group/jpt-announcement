export interface Schedule {
  id: string;
  title: string;
  description: string;
  audioId: string;
  timeOfDay: string; // "HH:MM"
  days: number[]; // 0=Minggu .. 6=Sabtu; kosong = setiap hari
  enabled: boolean;
  volume: number;
  lastTriggeredAt?: string;
  createdAt?: string;
  updatedAt?: string;
}

export interface AudioAsset {
  id: string;
  name: string;
  fileName: string;
  filePath: string;
  format: "mp3" | "wav" | string;
  sizeBytes: number;
  addedAt: string;
}

export interface AppConfig {
  startMinimized: boolean;
  autostartEnabled: boolean;
  defaultVolume: number;
  schedulerEnabled: boolean;
  language: string;
}

export interface LogEntry {
  timestamp: string;
  level: "INFO" | "WARN" | "ERROR" | string;
  source: string;
  message: string;
}

export interface PlatformCapabilities {
  platform: string;
  isMobile: boolean;
  isDesktop: boolean;
  supportsAutostart: boolean;
  supportsSystemTray: boolean;
  supportsBackgroundScheduler: boolean;
  supportsNativeFilePicker: boolean;
  supportedAudioFormats: string[];
}

export interface NextAnnouncementInfo {
  scheduleId: string;
  title: string;
  audioId: string;
  audioName: string;
  scheduledAt: string;
  remainingSec: number;
}

export interface PlaybackState {
  isPlaying: boolean;
  audioId?: string;
  audioName?: string;
  scheduleId?: string;
  startedAt?: string;
}

export interface DashboardState {
  platform: PlatformCapabilities;
  schedulerRunning: boolean;
  autostartEnabled: boolean;
  activeScheduleCount: number;
  totalScheduleCount: number;
  totalAudioCount: number;
  nextAnnouncement?: NextAnnouncementInfo;
  playback: PlaybackState;
}

export type NavTab = "dashboard" | "schedules" | "audio" | "settings";

export const HARI_INDONESIA: { value: number; short: string; label: string }[] = [
  { value: 1, short: "Sen", label: "Senin" },
  { value: 2, short: "Sel", label: "Selasa" },
  { value: 3, short: "Rab", label: "Rabu" },
  { value: 4, short: "Kam", label: "Kamis" },
  { value: 5, short: "Jum", label: "Jumat" },
  { value: 6, short: "Sab", label: "Sabtu" },
  { value: 0, short: "Min", label: "Minggu" }
];
