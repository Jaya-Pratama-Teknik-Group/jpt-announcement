<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from "vue";
import logoImg from "../logo.jpg";
import AlertModal from "./components/AlertModal.vue";
import ConfirmDeleteModal from "./components/ConfirmDeleteModal.vue";
import HugeIcon from "./components/HugeIcon.vue";
import PlaybackNotificationModal from "./components/PlaybackNotificationModal.vue";
import ScheduleModal from "./components/ScheduleModal.vue";
import { api } from "./api/bridge";
import type { HugeIconName } from "./icons/register";
import {
  HARI_INDONESIA,
  type AppConfig,
  type AudioAsset,
  type DashboardState,
  type LogEntry,
  type NavTab,
  type Schedule
} from "./types/models";

const activeTab = ref<NavTab>("dashboard");
const dashboard = ref<DashboardState | null>(null);
const schedules = ref<Schedule[]>([]);
const audios = ref<AudioAsset[]>([]);
const settings = ref<AppConfig>({
  startMinimized: true,
  autostartEnabled: false,
  defaultVolume: 85,
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
const fileInputRef = ref<HTMLInputElement | null>(null);
const scheduleImportInputRef = ref<HTMLInputElement | null>(null);
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

const navItems: { id: NavTab; label: string; short: string; icon: HugeIconName }[] = [
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

function formatDays(days: number[]): string {
  if (!days || days.length === 0 || days.length === 7) return "Setiap hari";
  return HARI_INDONESIA.filter((h) => days.includes(h.value))
    .map((h) => h.short)
    .join(", ");
}

function formatTimes(item: Schedule): string[] {
  if (item.timesOfDay && item.timesOfDay.length > 0) return item.timesOfDay;
  return item.timeOfDay ? [item.timeOfDay] : [];
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
}

function formatTime(iso?: string): string {
  if (!iso) return "-";
  return new Date(iso).toLocaleTimeString("id-ID", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false
  });
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

async function handleImportSchedules() {
  if (dashboard.value?.platform.supportsNativeFilePicker) {
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
    return;
  }
  scheduleImportInputRef.value?.click();
}

async function handleScheduleFileSelected(ev: Event) {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;

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

async function handlePickAudio() {
  if (dashboard.value?.platform.supportsNativeFilePicker) {
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
    return;
  }
  fileInputRef.value?.click();
}

async function handleWebFileSelected(ev: Event) {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;

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
    <!-- Sidebar Desktop / Layar Lebar -->
    <aside class="sidebar">
      <div class="brand brand-card">
        <div class="brand-logo-wrap">
          <img :src="logoImg" alt="Logo JPT Pengumuman Terjadwal" class="brand-logo" />
        </div>
        <div>
          <p class="brand-title">JPT Announce</p>
          <p class="brand-sub">Pengumuman Terjadwal</p>
        </div>
      </div>

      <nav class="sidebar-nav">
        <button
          v-for="item in navItems"
          :key="item.id"
          type="button"
          class="sidebar-btn"
          :class="[`tab-${item.id}`, { active: activeTab === item.id }]"
          @click="activeTab = item.id"
        >
          <span class="nav-icon-wrap">
            <HugeIcon :name="item.icon" :size="18" />
          </span>
          <span>{{ item.label }}</span>
        </button>
      </nav>

      <div style="margin-top: auto">
        <span
          class="badge"
          :class="dashboard?.schedulerRunning ? 'badge-success' : 'badge-muted'"
        >
          <HugeIcon name="activity-01" :size="14" />
          {{ dashboard?.schedulerRunning ? "Penjadwal Aktif" : "Penjadwal Nonaktif" }}
        </span>
      </div>
    </aside>

    <!-- Header Mobile -->
    <header class="mobile-header">
      <div class="brand">
        <div class="brand-logo-wrap">
          <img :src="logoImg" alt="Logo JPT Pengumuman Terjadwal" class="brand-logo" />
        </div>
        <div>
          <p class="brand-title">JPT Announce</p>
          <p class="brand-sub">
            {{ dashboard?.platform.platform === "android" ? "Mode Android" : "Pengumuman Terjadwal" }}
          </p>
        </div>
      </div>
      <span
        class="badge"
        :class="dashboard?.schedulerRunning ? 'badge-success' : 'badge-muted'"
      >
        {{ dashboard?.schedulerRunning ? "Aktif" : "Mati" }}
      </span>
    </header>

    <!-- Konten Utama -->
    <main class="main-content">
      <div
        v-if="dashboard && !dashboard.platform.supportsBackgroundScheduler"
        class="notice notice-warning"
      >
        <strong>Catatan Platform Android (Eksperimental):</strong> Jadwal berjalan saat aplikasi
        terbuka. Eksekusi audio otomatis di latar belakang saat layar terkunci memerlukan layanan
        latar belakang Android khusus dan belum aktif pada versi MVP ini.
      </div>

      <!-- TAB: DASBOR -->
      <section v-if="activeTab === 'dashboard'">
        <div class="page-header">
          <div>
            <h1 class="page-title">Dasbor Pengumuman</h1>
            <p class="page-subtitle">Pantau status penjadwal dan pemutaran audio terkini</p>
          </div>
          <div class="btn-row">
            <button class="btn btn-primary" type="button" @click="openCreateSchedule">
              <HugeIcon name="add-01" :size="17" />
              <span>Tambah Jadwal</span>
            </button>
            <button
              v-if="dashboard?.playback.isPlaying"
              class="btn btn-soft-primary"
              type="button"
              @click="handleTogglePauseAudio"
            >
              <HugeIcon :name="dashboard.playback.isPaused ? 'play' : 'pause'" :size="17" />
              <span>{{ dashboard.playback.isPaused ? "Lanjutkan" : "Jeda" }}</span>
            </button>
            <button
              v-if="dashboard?.playback.isPlaying"
              class="btn btn-danger"
              type="button"
              @click="handleStopAudio"
            >
              <HugeIcon name="stop" :size="17" />
              <span>Hentikan Audio</span>
            </button>
          </div>
        </div>

        <div class="grid-cards">
          <div class="card">
            <div class="card-head">
              <p class="card-title">Status Penjadwal</p>
              <span class="card-icon-chip chip-sky">
                <HugeIcon name="activity-01" :size="17" />
              </span>
            </div>
            <p class="metric-value">
              {{ dashboard?.schedulerRunning ? "Berjalan" : "Berhenti" }}
            </p>
            <span
              class="badge"
              :class="dashboard?.schedulerRunning ? 'badge-success' : 'badge-warning'"
            >
              {{ dashboard?.activeScheduleCount ?? 0 }} jadwal aktif
            </span>
          </div>

          <div class="card">
            <div class="card-head">
              <p class="card-title">Total Jadwal</p>
              <span class="card-icon-chip chip-lavender">
                <HugeIcon name="clock-01" :size="17" />
              </span>
            </div>
            <p class="metric-value">{{ dashboard?.totalScheduleCount ?? 0 }}</p>
            <span class="badge badge-muted">Tersimpan lokal</span>
          </div>

          <div class="card">
            <div class="card-head">
              <p class="card-title">Koleksi Audio</p>
              <span class="card-icon-chip chip-mint">
                <HugeIcon name="file-audio" :size="17" />
              </span>
            </div>
            <p class="metric-value">{{ dashboard?.totalAudioCount ?? 0 }}</p>
            <span class="badge badge-primary">.mp3, .mpeg &amp; .wav</span>
          </div>

          <div class="card">
            <div class="card-head">
              <p class="card-title">Status Pemutaran</p>
              <span class="card-icon-chip chip-amber">
                <HugeIcon name="audio-wave-01" :size="17" />
              </span>
            </div>
            <p class="metric-value" style="font-size: 1.15rem">
              {{
                dashboard?.playback.isPlaying
                  ? dashboard.playback.audioName || "Memutar..."
                  : "Siaga"
              }}
            </p>
            <span
              class="badge"
              :class="dashboard?.playback.isPlaying ? (dashboard.playback.isPaused ? 'badge-warning' : 'badge-primary') : 'badge-muted'"
            >
              {{
                dashboard?.playback.isPlaying
                  ? dashboard.playback.isPaused
                    ? "Dijeda"
                    : "Sedang diputar"
                  : "Tidak ada audio"
              }}
            </span>
          </div>
        </div>

        <div class="card">
          <div class="card-head">
            <p class="card-title">Pengumuman Berikutnya</p>
            <span class="card-icon-chip chip-lavender">
              <HugeIcon name="notification-02" :size="17" />
            </span>
          </div>
          <template v-if="dashboard?.nextAnnouncement">
            <h3 style="margin: 0 0 6px">{{ dashboard.nextAnnouncement.title }}</h3>
            <p class="page-subtitle" style="margin-bottom: 12px">
              Jam {{ new Date(dashboard.nextAnnouncement.scheduledAt).toLocaleTimeString("id-ID", { hour: "2-digit", minute: "2-digit", hour12: false }) }}
              &bull; Audio: {{ dashboard.nextAnnouncement.audioName || "-" }}
              &bull; Hitung mundur: <strong>{{ liveCountdownText }}</strong>
            </p>
            <div class="btn-row">
              <button
                class="btn btn-primary"
                type="button"
                @click="handlePlayScheduleNow(dashboard.nextAnnouncement.scheduleId)"
              >
                <HugeIcon name="play" :size="17" />
                <span>Putar Sekarang</span>
              </button>
              <button
                v-if="dashboard.playback.isPlaying"
                class="btn btn-danger"
                type="button"
                @click="handleStopAudio"
              >
                <HugeIcon name="stop" :size="17" />
                <span>Hentikan Audio</span>
              </button>
            </div>
          </template>
          <template v-else>
            <p class="page-subtitle" style="margin-bottom: 12px">
              Belum ada jadwal pengumuman aktif. Tambahkan berkas audio dan buat jadwal pertama Anda.
            </p>
            <button class="btn" type="button" @click="activeTab = 'schedules'">
              <HugeIcon name="calendar-03" :size="17" />
              <span>Kelola Jadwal</span>
            </button>
          </template>
        </div>
      </section>

      <!-- TAB: JADWAL -->
      <section v-if="activeTab === 'schedules'">
        <div class="page-header">
          <div>
            <h1 class="page-title">Jadwal Pengumuman</h1>
            <p class="page-subtitle">Atur waktu dan hari pemutaran audio otomatis</p>
          </div>
          <div class="btn-row">
            <button
              class="btn"
              type="button"
              :disabled="busy || schedules.length === 0"
              @click="handleExportSchedules"
            >
              <HugeIcon name="file-export" :size="17" />
              <span>Ekspor</span>
            </button>
            <button class="btn" type="button" :disabled="busy" @click="handleImportSchedules">
              <HugeIcon name="file-import" :size="17" />
              <span>Impor</span>
            </button>
            <button
              class="btn btn-primary"
              type="button"
              :disabled="audios.length === 0"
              @click="openCreateSchedule"
            >
              <HugeIcon name="add-01" :size="17" />
              <span>Tambah Jadwal</span>
            </button>
          </div>
          <input
            ref="scheduleImportInputRef"
            type="file"
            accept=".json,application/json"
            style="display: none"
            @change="handleScheduleFileSelected"
          />
        </div>

        <div v-if="audios.length === 0" class="notice notice-warning">
          Tambahkan minimal satu berkas audio di menu <strong>Audio</strong> sebelum membuat atau
          mengaktifkan jadwal.
        </div>

        <div v-if="schedulesWithoutAudioCount > 0" class="notice notice-warning">
          Ada <strong>{{ schedulesWithoutAudioCount }} jadwal</strong> yang belum memiliki berkas
          audio (hasil impor). Klik <strong>Pilih Audio</strong> pada jadwal tersebut agar dapat
          diaktifkan.
        </div>

        <div v-if="schedules.length === 0" class="card">
          <p class="page-subtitle">Belum ada jadwal pengumuman yang tersimpan.</p>
        </div>

        <!-- Tabel Desktop -->
        <div v-if="schedules.length > 0" class="card desktop-only">
          <table class="data-table">
            <thead>
              <tr>
                <th>Judul</th>
                <th>Jam</th>
                <th>Hari</th>
                <th>Audio</th>
                <th>Volume</th>
                <th>Status</th>
                <th>Aksi</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in schedules" :key="item.id">
                <td>
                  <strong>{{ item.title }}</strong>
                  <div v-if="item.description" class="page-subtitle">{{ item.description }}</div>
                </td>
                <td>
                  <div class="time-pills">
                    <span v-for="t in formatTimes(item)" :key="t" class="time-pill">{{ t }}</span>
                  </div>
                </td>
                <td>{{ formatDays(item.days) }}</td>
                <td>
                  <span v-if="hasValidAudio(item)">{{ audioNameMap.get(item.audioId) }}</span>
                  <span v-else class="badge badge-warning">
                    <HugeIcon name="alert-02" :size="14" />
                    <span>Perlu pilih audio</span>
                  </span>
                </td>
                <td>{{ item.volume }}%</td>
                <td>
                  <span class="badge" :class="item.enabled ? 'badge-success' : 'badge-muted'">
                    {{ item.enabled ? "Aktif" : "Nonaktif" }}
                  </span>
                </td>
                <td>
                  <div class="btn-row">
                    <button
                      class="btn btn-sm"
                      type="button"
                      :disabled="!hasValidAudio(item)"
                      @click="handlePlayScheduleNow(item.id)"
                    >
                      <HugeIcon name="play" :size="15" />
                      <span>Putar</span>
                    </button>
                    <button
                      class="btn btn-sm"
                      type="button"
                      :disabled="!item.enabled && !hasValidAudio(item)"
                      :title="
                        !item.enabled && !hasValidAudio(item)
                          ? 'Pilih audio terlebih dahulu sebelum mengaktifkan'
                          : ''
                      "
                      @click="handleToggleSchedule(item)"
                    >
                      <HugeIcon name="power" :size="15" />
                      <span>{{ item.enabled ? "Matikan" : "Aktifkan" }}</span>
                    </button>
                    <button
                      class="btn btn-sm"
                      :class="!hasValidAudio(item) ? 'btn-primary' : ''"
                      type="button"
                      @click="openEditSchedule(item)"
                    >
                      <HugeIcon name="edit-02" :size="15" />
                      <span>{{ hasValidAudio(item) ? "Ubah" : "Pilih Audio" }}</span>
                    </button>
                    <button
                      class="btn btn-sm btn-danger"
                      type="button"
                      @click="handleDeleteSchedule(item)"
                    >
                      <HugeIcon name="delete-02" :size="15" />
                      <span>Hapus</span>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Kartu Ringkas Mobile -->
        <div v-if="schedules.length > 0" class="mobile-list">
          <div v-for="item in schedules" :key="item.id" class="card">
            <div style="display: flex; justify-content: space-between; align-items: start; gap: 8px">
              <div>
                <h3 style="margin: 0">{{ item.title }}</h3>
                <div class="time-pills" style="margin: 6px 0 4px">
                  <span v-for="t in formatTimes(item)" :key="t" class="time-pill">{{ t }}</span>
                </div>
                <p class="page-subtitle">{{ formatDays(item.days) }}</p>
              </div>
              <span class="badge" :class="item.enabled ? 'badge-success' : 'badge-muted'">
                {{ item.enabled ? "Aktif" : "Nonaktif" }}
              </span>
            </div>
            <p class="page-subtitle" style="margin: 8px 0 12px">
              Audio:
              <template v-if="hasValidAudio(item)">{{ audioNameMap.get(item.audioId) }}</template>
              <span v-else class="badge badge-warning">Perlu pilih audio</span>
              &bull; Volume {{ item.volume }}%
            </p>
            <div class="btn-row">
              <button
                class="btn btn-sm btn-primary"
                type="button"
                :disabled="!hasValidAudio(item)"
                @click="handlePlayScheduleNow(item.id)"
              >
                <HugeIcon name="play" :size="15" />
                <span>Putar</span>
              </button>
              <button
                class="btn btn-sm"
                type="button"
                :disabled="!item.enabled && !hasValidAudio(item)"
                @click="handleToggleSchedule(item)"
              >
                <HugeIcon name="power" :size="15" />
                <span>{{ item.enabled ? "Nonaktifkan" : "Aktifkan" }}</span>
              </button>
              <button
                class="btn btn-sm"
                :class="!hasValidAudio(item) ? 'btn-primary' : ''"
                type="button"
                @click="openEditSchedule(item)"
              >
                <HugeIcon name="edit-02" :size="15" />
                <span>{{ hasValidAudio(item) ? "Ubah" : "Pilih Audio" }}</span>
              </button>
              <button class="btn btn-sm btn-danger" type="button" @click="handleDeleteSchedule(item)">
                <HugeIcon name="delete-02" :size="15" />
                <span>Hapus</span>
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- TAB: AUDIO -->
      <section v-if="activeTab === 'audio'">
        <div class="page-header">
          <div>
            <h1 class="page-title">Manajemen Audio</h1>
            <p class="page-subtitle">Unggah dan uji berkas audio lokal (.mp3, .mpeg, .mpga, atau .wav)</p>
          </div>
          <div class="btn-row">
            <button class="btn btn-primary" type="button" :disabled="busy" @click="handlePickAudio">
              <HugeIcon name="add-01" :size="17" />
              <span>Tambah Audio</span>
            </button>
            <button
              v-if="dashboard?.playback.isPlaying"
              class="btn btn-danger"
              type="button"
              @click="handleStopAudio"
            >
              <HugeIcon name="stop" :size="17" />
              <span>Hentikan</span>
            </button>
          </div>
          <input
            ref="fileInputRef"
            type="file"
            accept=".mp3,.mpeg,.mpga,.wav,audio/mpeg,audio/wav"
            style="display: none"
            @change="handleWebFileSelected"
          />
        </div>

        <div v-if="audios.length === 0" class="card">
          <p class="page-subtitle">
            Belum ada berkas audio. Klik <strong>Tambah Audio</strong> untuk memilih berkas .mp3,
            .mpeg, atau .wav dari perangkat Anda.
          </p>
        </div>

        <div v-else class="card audio-list-card">
          <div v-for="asset in audios" :key="asset.id" class="audio-row">
            <div class="audio-row-main">
              <span class="card-icon-chip chip-mint">
                <HugeIcon name="file-audio" :size="18" />
              </span>
              <div class="audio-row-info">
                <div class="audio-row-title-line">
                  <strong class="audio-row-title">{{ asset.name }}</strong>
                  <span class="badge badge-primary">.{{ asset.format }}</span>
                </div>
                <p class="page-subtitle audio-row-meta">
                  {{ asset.fileName }} &bull; {{ formatSize(asset.sizeBytes) }}
                </p>
              </div>
            </div>
            <div class="btn-row audio-row-actions">
              <button
                v-if="dashboard?.playback.isPlaying && dashboard.playback.audioId === asset.id"
                class="btn btn-sm btn-danger"
                type="button"
                @click="handleStopAudio"
              >
                <HugeIcon name="stop" :size="15" />
                <span>Hentikan</span>
              </button>
              <button
                v-else
                class="btn btn-sm btn-primary"
                type="button"
                @click="handlePlayAudio(asset.id)"
              >
                <HugeIcon name="play" :size="15" />
                <span>Uji Putar</span>
              </button>
              <button class="btn btn-sm btn-danger" type="button" @click="handleDeleteAudio(asset)">
                <HugeIcon name="delete-02" :size="15" />
                <span>Hapus</span>
              </button>
            </div>
          </div>
        </div>
      </section>

      <!-- TAB: PENGATURAN -->
      <section v-if="activeTab === 'settings'">
        <div class="page-header">
          <div>
            <h1 class="page-title">Pengaturan Aplikasi</h1>
            <p class="page-subtitle">Konfigurasi perilaku penjadwal, audio, dan startup platform</p>
          </div>
        </div>

        <div class="card" style="margin-bottom: 16px">
          <div class="switch-row">
            <div>
              <strong>Aktifkan Penjadwal Otomatis</strong>
              <p class="page-subtitle">Jalankan pemutaran audio otomatis sesuai jadwal aktif</p>
            </div>
            <input v-model="settings.schedulerEnabled" class="switch-input" type="checkbox" />
          </div>

          <div v-if="dashboard?.platform.supportsAutostart" class="switch-row">
            <div>
              <strong>Jalankan otomatis saat Windows dimulai</strong>
              <p class="page-subtitle">
                Status saat ini:
                <strong>{{ settings.autostartEnabled ? "Aktif" : "Nonaktif" }}</strong> &bull;
                Mulai tersembunyi ke system tray saat login Windows (tanpa hak admin)
              </p>
            </div>
            <button
              class="btn btn-sm"
              :class="settings.autostartEnabled ? 'btn-primary' : ''"
              type="button"
              :disabled="busy"
              @click="handleToggleAutostart"
            >
              {{ settings.autostartEnabled ? "Aktif (Matikan)" : "Aktifkan" }}
            </button>
          </div>

          <div v-if="dashboard?.platform.supportsSystemTray" class="switch-row">
            <div>
              <strong>Sembunyikan ke Tray Saat Login</strong>
              <p class="page-subtitle">
                Saat dijalankan otomatis di login Windows, aplikasi mulai tersembunyi di system
                tray; menutup jendela utama juga akan menyembunyikannya ke tray
              </p>
            </div>
            <input v-model="settings.startMinimized" class="switch-input" type="checkbox" />
          </div>

          <div class="field" style="margin-top: 14px">
            <label for="cfg-vol" style="display: inline-flex; align-items: center; gap: 6px">
              <HugeIcon name="volume-high" :size="16" />
              <span>Volume Bawaan: {{ settings.defaultVolume }}%</span>
            </label>
            <input id="cfg-vol" v-model.number="settings.defaultVolume" type="range" min="0" max="100" />
          </div>

          <div class="btn-row" style="margin-top: 16px">
            <button class="btn btn-primary" type="button" :disabled="busy" @click="handleSaveSettings">
              <HugeIcon name="checkmark-circle-02" :size="17" />
              <span>Simpan Pengaturan</span>
            </button>
          </div>
        </div>

        <div class="card">
          <h3 style="margin: 0 0 12px">Log Aktivitas Terbaru</h3>
          <p v-if="logs.length === 0" class="page-subtitle">Belum ada catatan aktivitas.</p>
          <div v-else class="form-stack">
            <div
              v-for="(entry, idx) in logs.slice(0, 20)"
              :key="idx"
              style="border-bottom: 1px solid var(--border); padding-bottom: 8px"
            >
              <div style="display: flex; justify-content: space-between; gap: 8px">
                <span
                  class="badge"
                  :class="
                    entry.level === 'ERROR'
                      ? 'badge-danger'
                      : entry.level === 'WARN'
                        ? 'badge-warning'
                        : 'badge-muted'
                  "
                >
                  {{ entry.level }} &bull; {{ entry.source }}
                </span>
                <span class="page-subtitle">{{ formatTime(entry.timestamp) }}</span>
              </div>
              <p style="margin: 4px 0 0; font-size: 0.9rem">{{ entry.message }}</p>
            </div>
          </div>
        </div>
      </section>
    </main>

    <!-- Bottom Navigation Mobile -->
    <nav class="bottom-nav">
      <button
        v-for="item in navItems"
        :key="item.id"
        type="button"
        class="bottom-nav-btn"
        :class="[`tab-${item.id}`, { active: activeTab === item.id }]"
        @click="activeTab = item.id"
      >
        <span class="bottom-nav-icon">
          <HugeIcon :name="item.icon" :size="19" />
        </span>
        <span>{{ item.short }}</span>
      </button>
    </nav>

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
