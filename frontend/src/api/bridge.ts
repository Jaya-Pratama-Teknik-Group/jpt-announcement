import { Call, Events } from "@wailsio/runtime";
import type {
  AppConfig,
  AudioAsset,
  DashboardState,
  LogEntry,
  NavTab,
  Schedule
} from "../types/models";

const SERVICE_PREFIX = "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/announcement.Service";

function invoke<T>(method: string, ...args: unknown[]): Promise<T> {
  return Call.ByName(`${SERVICE_PREFIX}.${method}`, ...args) as Promise<T>;
}

export const api = {
  getDashboardState(): Promise<DashboardState> {
    return invoke<DashboardState>("GetDashboardState");
  },
  listSchedules(): Promise<Schedule[]> {
    return invoke<Schedule[]>("ListSchedules");
  },
  saveSchedule(item: Schedule): Promise<Schedule> {
    return invoke<Schedule>("SaveSchedule", item);
  },
  toggleSchedule(id: string, enabled: boolean): Promise<Schedule> {
    return invoke<Schedule>("ToggleSchedule", id, enabled);
  },
  deleteSchedule(id: string): Promise<void> {
    return invoke<void>("DeleteSchedule", id);
  },
  listAudioFiles(): Promise<AudioAsset[]> {
    return invoke<AudioAsset[]>("ListAudioFiles");
  },
  pickAndImportAudio(): Promise<AudioAsset | null> {
    return invoke<AudioAsset | null>("PickAndImportAudio");
  },
  importAudioBase64(fileName: string, displayName: string, base64Data: string): Promise<AudioAsset> {
    return invoke<AudioAsset>("ImportAudioBase64", fileName, displayName, base64Data);
  },
  deleteAudioFile(id: string): Promise<void> {
    return invoke<void>("DeleteAudioFile", id);
  },
  playAudio(audioId: string, volume: number): Promise<void> {
    return invoke<void>("PlayAudio", audioId, volume);
  },
  playScheduleNow(scheduleId: string): Promise<void> {
    return invoke<void>("PlayScheduleNow", scheduleId);
  },
  stopAudio(): Promise<void> {
    return invoke<void>("StopAudio");
  },
  getSettings(): Promise<AppConfig> {
    return invoke<AppConfig>("GetSettings");
  },
  updateSettings(cfg: AppConfig): Promise<AppConfig> {
    return invoke<AppConfig>("UpdateSettings", cfg);
  },
  setAutostartEnabled(enabled: boolean): Promise<boolean> {
    return invoke<boolean>("SetAutostartEnabled", enabled);
  },
  listLogs(): Promise<LogEntry[]> {
    return invoke<LogEntry[]>("ListLogs");
  },
  onStateUpdated(handler: (state: DashboardState) => void): () => void {
    return Events.On("state:updated", (ev) => {
      const payload = Array.isArray(ev.data) ? ev.data[0] : ev.data;
      if (payload) {
        handler(payload as DashboardState);
      }
    });
  },
  onNavigate(handler: (tab: NavTab) => void): () => void {
    return Events.On("app:navigate", (ev) => {
      const payload = Array.isArray(ev.data) ? ev.data[0] : ev.data;
      if (typeof payload === "string") {
        handler(payload as NavTab);
      }
    });
  }
};
