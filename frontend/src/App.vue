<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import AlertModal from "./components/AlertModal.vue";
import AppTopBar from "./components/AppTopBar.vue";
import BottomDock, { type DockNavItem } from "./components/BottomDock.vue";
import ConfirmDeleteModal from "./components/ConfirmDeleteModal.vue";
import PlaybackNotificationModal from "./components/PlaybackNotificationModal.vue";
import ScheduleModal from "./components/ScheduleModal.vue";
import AudioView from "./views/AudioView.vue";
import DashboardView from "./views/DashboardView.vue";
import SchedulesView from "./views/SchedulesView.vue";
import SettingsView from "./views/SettingsView.vue";
import { api } from "./api/bridge";
import { formatDays, formatSize, formatTimes } from "./utils/formatters";
import type {
  AppConfig,
  AudioAsset,
  DashboardState,
  LogEntry,
  NavTab,
  Schedule
} from "./types/models";

const activeTab = ref<NavTab>("dashboard");
const dashboard = ref<DashboardState | null>(null);
const schedules = ref<Schedule[]>([]);
const audios = ref<AudioAsset[]>([]);
const settings = ref<AppConfig>({
  startMinimized: true,
  autostartEnabled: false,
  defaultVolume: 85,
  duckOtherAudio: true,
  duckVolumePercent: 25,
  liveAnnouncementEnabled: false,
  schedulerEnabled: true,
  language: "id"
});
const logs = ref<LogEntry[]>([]);

const alertDialog = ref<{
  open: boolean;
  kind: "info" | "error";
  message: string;
  nonce: number;
}>({
  open: false,
  kind: "info",
  message: "",
  nonce: 0
});
const busy = ref(false);
const modalOpen = ref(false);
const editingSchedule = ref<Schedule | null>(null);
const deleteDialog = ref<{
  open: boolean;
  kind: "schedule" | "audio";
  id: string;
  title: string;
  itemName: string;
  itemDetail: string;
  description: string;
}>({
  open: false,
  kind: "schedule",
  id: "",
  title: "",
  itemName: "",
  itemDetail: "",
  description: ""
});
const countdownTick = ref(Date.now());
const dismissedPlaybackKey = ref("");

const currentPlaybackKey = computed(() => {
  const pb = dashboard.value?.playback;
  if (!pb?.isPlaying || !pb.scheduleId) return "";
  return `${pb.scheduleId}:${pb.startedAt ?? ""}`;
});

const showPlaybackNotification = computed(() => {
  if (!currentPlaybackKey.value) return false;
  return dismissedPlaybackKey.value !== currentPlaybackKey.value;
});

function dismissPlaybackNotification() {
  if (currentPlaybackKey.value) {
    dismissedPlaybackKey.value = currentPlaybackKey.value;
  }
}

let unsubState: (() => void) | null = null;
let unsubNav: (() => void) | null = null;
let timerId: number | null = null;

const navItems: DockNavItem[] = [
  { id: "dashboard", label: "Dasbor", short: "Dasbor", icon: "home-03" },
  { id: "schedules", label: "Jadwal", short: "Jadwal", icon: "calendar-03" },
  { id: "audio", label: "Audio", short: "Audio", icon: "music-note-03" },
  { id: "settings", label: "Pengaturan", short: "Atur", icon: "settings-02" }
];

const audioNameMap = computed(() => {
  const map = new Map<string, string>();
  for (const a of audios.value) {
    map.set(a.id, a.name);
  }
  return map;
});

function hasValidAudio(item: Schedule): boolean {
  return Boolean(item.audioId && audioNameMap.value.has(item.audioId));
}

const schedulesWithoutAudioCount = computed(() =>
  schedules.value.filter((item) => !hasValidAudio(item)).length
);

const liveCountdownText = computed(() => {
  const next = dashboard.value?.nextAnnouncement;
  if (!next) return "-";
  const targetMs = new Date(next.scheduledAt).getTime();
  const diffSec = Math.max(0, Math.floor((targetMs - countdownTick.value) / 1000));
  const hrs = Math.floor(diffSec / 3600);
  const mins = Math.floor((diffSec % 3600) / 60);
  const secs = diffSec % 60;
  if (hrs > 0) return `${hrs} jam ${mins} menit`;
  if (mins > 0) return `${mins} menit ${secs} detik`;
  return `${secs} detik lagi`;
});

async function refreshAll() {
  try {
    const [d, s, a, cfg, l] = await Promise.all([
      api.getDashboardState(),
      api.listSchedules(),
      api.listAudioFiles(),
      api.getSettings(),
      api.listLogs()
    ]);
    dashboard.value = d;
    schedules.value = s;
    audios.value = a;
    settings.value = cfg;
    logs.value = l;
  } catch (err) {
    showError(err);
  }
}

function showError(err: unknown) {
  alertDialog.value = {
    open: true,
    kind: "error",
    message: err instanceof Error ? err.message : String(err),
    nonce: alertDialog.value.nonce + 1
  };
}

function showInfo(msg: string) {
  alertDialog.value = {
    open: true,
    kind: "info",
    message: msg,
    nonce: alertDialog.value.nonce + 1
  };
}

function closeAlertDialog() {
  alertDialog.value.open = false;
}

function openCreateSchedule() {
  editingSchedule.value = null;
  modalOpen.value = true;
}

function openEditSchedule(item: Schedule) {
  editingSchedule.value = item;
  modalOpen.value = true;
}

async function handleSaveSchedule(item: Schedule) {
  busy.value = true;
  try {
    await api.saveSchedule(item);
    modalOpen.value = false;
    showInfo(`Jadwal "${item.title}" berhasil disimpan.`);
    await refreshAll();
  } catch (err) {
    showError(err);
  } finally {
    busy.value = false;
  }
}

async function handleToggleSchedule(item: Schedule) {
  try {
    await api.toggleSchedule(item.id, !item.enabled);
    await refreshAll();
  } catch (err) {
    showError(err);
  }
}

function handleDeleteSchedule(item: Schedule) {
  deleteDialog.value = {
    open: true,
    kind: "schedule",
    id: item.id,
    title: "Hapus Jadwal Pengumuman?",
    itemName: item.title,
    itemDetail: `${formatTimes(item).join(", ")} • ${formatDays(item.days)}`,
    description: "Jadwal ini akan dihapus permanen dari daftar pengumuman otomatis."
  };
}

async function handleExportSchedules() {
  busy.value = true;
  try {
    if (dashboard.value?.platform.supportsNativeFilePicker) {
      const savedPath = await api.exportSchedulesToDialog();
      if (savedPath) {
        showInfo(`Konfigurasi jadwal berhasil diekspor ke ${savedPath}.`);
      }
      return;
    }
    const rawJSON = await api.exportSchedulesJSON();
    const blob = new Blob([rawJSON], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `jpt-announce-jadwal-${new Date().toISOString().slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(url);
    showInfo("Konfigurasi jadwal berhasil diekspor.");
  } catch (err) {
    showError(err);
  } finally {
    busy.value = false;
  }
}

async function handleImportSchedulesNative() {
  busy.value = true;
  try {
    const imported = await api.pickAndImportSchedules();
    if (imported && imported.length > 0) {
      showInfo(
        `${imported.length} jadwal berhasil diimpor (status Nonaktif). Pilih audio untuk setiap jadwal sebelum mengaktifkannya.`
      );
      await refreshAll();
    }
  } catch (err) {
    showError(err);
  } finally {
    busy.value = false;
  }
}

async function handleScheduleFileSelected(file: File) {
  busy.value = true;
  try {
    const rawJSON = await file.text();
    const imported = await api.importSchedulesJSON(rawJSON);
    showInfo(
      `${imported.length} jadwal berhasil diimpor (status Nonaktif). Pilih audio untuk setiap jadwal sebelum mengaktifkannya.`
    );
    await refreshAll();
  } catch (err) {
    showError(err);
  } finally {
    busy.value = false;
  }
}

async function handlePickAudioNative() {
  busy.value = true;
  try {
    const added = await api.pickAndImportAudio();
    if (added) {
      showInfo(`Audio "${added.name}" berhasil ditambahkan.`);
      await refreshAll();
    }
  } catch (err) {
    showError(err);
  } finally {
    busy.value = false;
  }
}

async function handleWebFileSelected(file: File) {
  busy.value = true;
  try {
    const buffer = await file.arrayBuffer();
    const bytes = new Uint8Array(buffer);
    let binary = "";
    const chunkSize = 0x8000;
    for (let i = 0; i < bytes.length; i += chunkSize) {
      binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize));
    }
    const base64 = btoa(binary);
    const added = await api.importAudioBase64(file.name, "", base64);
    showInfo(`Audio "${added.name}" berhasil ditambahkan.`);
    await refreshAll();
  } catch (err) {
    showError(err);
  } finally {
    busy.value = false;
  }
}

function handleDeleteAudio(asset: AudioAsset) {
  deleteDialog.value = {
    open: true,
    kind: "audio",
    id: asset.id,
    title: "Hapus Berkas Audio?",
    itemName: asset.name,
    itemDetail: `${asset.fileName} • ${formatSize(asset.sizeBytes)}`,
    description:
      "Berkas audio akan dihapus dari penyimpanan aplikasi dan tidak dapat dipulihkan."
  };
}

async function confirmDeleteTarget() {
  if (!deleteDialog.value.open || !deleteDialog.value.id) return;
  const { kind, id, itemName } = deleteDialog.value;
  busy.value = true;
  try {
    if (kind === "schedule") {
      await api.deleteSchedule(id);
      showInfo(`Jadwal "${itemName}" dihapus.`);
    } else {
      await api.deleteAudioFile(id);
      showInfo(`Audio "${itemName}" dihapus.`);
    }
    deleteDialog.value.open = false;
    await refreshAll();
  } catch (err) {
    deleteDialog.value.open = false;
    showError(err);
  } finally {
    busy.value = false;
  }
}

async function handlePlayAudio(audioId: string) {
  try {
    await api.playAudio(audioId, settings.value.defaultVolume);
    await refreshAll();
  } catch (err) {
    showError(err);
  }
}

async function handlePreviewScheduleAudio(audioId: string, volume: number) {
  try {
    await api.playAudio(audioId, volume);
    await refreshAll();
  } catch (err) {
    showError(err);
  }
}

async function handlePlayScheduleNow(scheduleId: string) {
  try {
    await api.playScheduleNow(scheduleId);
    await refreshAll();
  } catch (err) {
    showError(err);
  }
}

async function handleTogglePauseAudio() {
  try {
    await api.togglePauseAudio();
    await refreshAll();
  } catch (err) {
    showError(err);
  }
}

async function handleStopAudio() {
  try {
    await api.stopAudio();
    await refreshAll();
  } catch (err) {
    showError(err);
  }
}

async function handleToggleLiveAnnouncement() {
  try {
    if (dashboard.value?.playback.isLive) {
      await api.stopLiveAnnouncement();
    } else {
      await api.startLiveAnnouncement();
    }
    await refreshAll();
  } catch (err) {
    showError(err);
  }
}

async function handleSaveSettings() {
  busy.value = true;
  try {
    settings.value = await api.updateSettings({ ...settings.value });
    showInfo("Pengaturan berhasil disimpan.");
    await refreshAll();
  } catch (err) {
    showError(err);
  } finally {
    busy.value = false;
  }
}

async function handleToggleAutostart() {
  busy.value = true;
  try {
    const enabled = await api.setAutostartEnabled(!settings.value.autostartEnabled);
    settings.value.autostartEnabled = enabled;
    showInfo(
      enabled
        ? "Aplikasi akan berjalan otomatis saat Windows dimulai."
        : "Jalankan otomatis saat Windows dimulai dinonaktifkan."
    );
    await refreshAll();
  } catch (err) {
    showError(err);
  } finally {
    busy.value = false;
  }
}

onMounted(async () => {
  await refreshAll();
  unsubState = api.onStateUpdated((nextState) => {
    dashboard.value = nextState;
    void api.listLogs().then((l) => {
      logs.value = l;
    });
  });
  unsubNav = api.onNavigate((tab) => {
    activeTab.value = tab;
  });
  timerId = window.setInterval(() => {
    countdownTick.value = Date.now();
  }, 1000);
});

onUnmounted(() => {
  unsubState?.();
  unsubNav?.();
  if (timerId !== null) window.clearInterval(timerId);
});
</script>

<template>
  <div class="app-shell">
    <AppTopBar
      :scheduler-running="Boolean(dashboard?.schedulerRunning)"
      :platform-name="dashboard?.platform.platform"
      :playback="dashboard?.playback"
    />

    <main class="main-content">
      <div
        v-if="dashboard && !dashboard.platform.supportsBackgroundScheduler"
        class="notice notice-warning"
      >
        <strong>Catatan Platform Android (Eksperimental):</strong> Jadwal berjalan saat aplikasi
        terbuka. Eksekusi audio otomatis di latar belakang saat layar terkunci memerlukan layanan
        latar belakang Android khusus dan belum aktif pada versi MVP ini.
      </div>

      <Transition name="view-fade" mode="out-in">
        <DashboardView
          v-if="activeTab === 'dashboard'"
          key="dashboard"
          :dashboard="dashboard"
          :settings="settings"
          :live-countdown-text="liveCountdownText"
          @create-schedule="openCreateSchedule"
          @toggle-live="handleToggleLiveAnnouncement"
          @toggle-pause="handleTogglePauseAudio"
          @stop-audio="handleStopAudio"
          @play-schedule-now="handlePlayScheduleNow"
          @navigate-schedules="activeTab = 'schedules'"
        />

        <SchedulesView
          v-else-if="activeTab === 'schedules'"
          key="schedules"
          :schedules="schedules"
          :audios="audios"
          :audio-name-map="audioNameMap"
          :schedules-without-audio-count="schedulesWithoutAudioCount"
          :supports-native-file-picker="Boolean(dashboard?.platform.supportsNativeFilePicker)"
          :busy="busy"
          @export-schedules="handleExportSchedules"
          @import-schedules="handleImportSchedulesNative"
          @import-schedule-file="handleScheduleFileSelected"
          @create-schedule="openCreateSchedule"
          @play-schedule-now="handlePlayScheduleNow"
          @toggle-schedule="handleToggleSchedule"
          @edit-schedule="openEditSchedule"
          @delete-schedule="handleDeleteSchedule"
        />

        <AudioView
          v-else-if="activeTab === 'audio'"
          key="audio"
          :audios="audios"
          :playback="dashboard?.playback"
          :supports-native-file-picker="Boolean(dashboard?.platform.supportsNativeFilePicker)"
          :busy="busy"
          @pick-audio="handlePickAudioNative"
          @import-audio-file="handleWebFileSelected"
          @play-audio="handlePlayAudio"
          @stop-audio="handleStopAudio"
          @delete-audio="handleDeleteAudio"
        />

        <SettingsView
          v-else
          key="settings"
          v-model="settings"
          :platform="dashboard?.platform"
          :logs="logs"
          :busy="busy"
          @save-settings="handleSaveSettings"
          @toggle-autostart="handleToggleAutostart"
        />
      </Transition>
    </main>

    <BottomDock :items="navItems" :active-tab="activeTab" @select="activeTab = $event" />

    <ScheduleModal
      :open="modalOpen"
      :initial="editingSchedule"
      :audios="audios"
      :default-volume="settings.defaultVolume"
      :playback="dashboard?.playback"
      @close="modalOpen = false"
      @save="handleSaveSchedule"
      @preview-audio="handlePreviewScheduleAudio"
      @stop-audio="handleStopAudio"
    />

    <ConfirmDeleteModal
      :open="deleteDialog.open"
      :title="deleteDialog.title"
      :item-name="deleteDialog.itemName"
      :item-detail="deleteDialog.itemDetail"
      :description="deleteDialog.description"
      :busy="busy"
      @close="deleteDialog.open = false"
      @confirm="confirmDeleteTarget"
    />

    <PlaybackNotificationModal
      :open="showPlaybackNotification"
      :playback="dashboard?.playback"
      @toggle-pause="handleTogglePauseAudio"
      @stop="handleStopAudio"
      @dismiss="dismissPlaybackNotification"
    />

    <AlertModal
      :open="alertDialog.open"
      :kind="alertDialog.kind"
      :message="alertDialog.message"
      :nonce="alertDialog.nonce"
      :auto-close-seconds="10"
      @close="closeAlertDialog"
    />
  </div>
</template>
