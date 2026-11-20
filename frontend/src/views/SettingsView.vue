<script setup lang="ts">
import HugeIcon from "../components/HugeIcon.vue";
import { useTheme, type ThemePreference } from "../utils/theme";
import type { AppConfig, LogEntry, PlatformCapabilities } from "../types/models";

const settings = defineModel<AppConfig>({ required: true });

defineProps<{
  platform?: PlatformCapabilities;
  logs: LogEntry[];
  busy: boolean;
}>();

const emit = defineEmits<{
  (e: "save-settings"): void;
  (e: "toggle-autostart"): void;
}>();

const { themePreference, setThemePreference } = useTheme();

const themeOptions: { value: ThemePreference; label: string }[] = [
  { value: "light", label: "Terang" },
  { value: "dark", label: "Gelap" },
  { value: "system", label: "Sistem" }
];

function formatTime(iso?: string): string {
  if (!iso) return "-";
  return new Date(iso).toLocaleTimeString("id-ID", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false
  });
}
</script>

<template>
  <section>
    <div class="page-header">
      <div>
        <h1 class="page-title">Pengaturan Aplikasi</h1>
        <p class="page-subtitle">Konfigurasi perilaku penjadwal, audio, dan startup platform</p>
      </div>
    </div>

    <div class="card" style="margin-bottom: 16px">
      <p class="settings-section-title">Tampilan</p>

      <div class="switch-row" style="flex-wrap: wrap">
        <div>
          <strong>Tema Aplikasi</strong>
          <p class="page-subtitle">Pilih tampilan terang, gelap, atau ikuti tema sistem</p>
        </div>
        <div class="theme-switcher" role="radiogroup" aria-label="Pilih tema aplikasi">
          <button
            v-for="opt in themeOptions"
            :key="opt.value"
            type="button"
            role="radio"
            class="theme-option"
            :class="{ active: themePreference === opt.value }"
            :aria-checked="themePreference === opt.value"
            @click="setThemePreference(opt.value)"
          >
            {{ opt.label }}
          </button>
        </div>
      </div>

      <p class="settings-section-title" style="margin-top: 18px">Umum &amp; Startup</p>

      <div class="switch-row">
        <div>
          <strong>Aktifkan Penjadwal Otomatis</strong>
          <p class="page-subtitle">Jalankan pemutaran audio otomatis sesuai jadwal aktif</p>
        </div>
        <label class="toggle-switch">
          <input v-model="settings.schedulerEnabled" type="checkbox" />
          <span class="toggle-track"></span>
        </label>
      </div>

      <div v-if="platform?.supportsAutostart" class="switch-row">
        <div>
          <strong>Jalankan otomatis saat Windows dimulai</strong>
          <p class="page-subtitle">
            Status saat ini:
            <strong>{{ settings.autostartEnabled ? "Aktif" : "Nonaktif" }}</strong> &bull; Mulai
            tersembunyi ke system tray saat login Windows (tanpa hak admin)
          </p>
        </div>
        <button
          class="btn btn-sm"
          :class="settings.autostartEnabled ? 'btn-soft-primary' : ''"
          type="button"
          :disabled="busy"
          @click="emit('toggle-autostart')"
        >
          {{ settings.autostartEnabled ? "Aktif (Matikan)" : "Aktifkan" }}
        </button>
      </div>

      <div v-if="platform?.supportsSystemTray" class="switch-row">
        <div>
          <strong>Sembunyikan ke Tray Saat Login</strong>
          <p class="page-subtitle">
            Saat dijalankan otomatis di login Windows, aplikasi mulai tersembunyi di system tray;
            menutup jendela utama juga akan menyembunyikannya ke tray
          </p>
        </div>
        <label class="toggle-switch">
          <input v-model="settings.startMinimized" type="checkbox" />
          <span class="toggle-track"></span>
        </label>
      </div>

      <p class="settings-section-title" style="margin-top: 18px">Audio &amp; Siaran</p>

      <div class="field" style="padding: 8px 0 12px">
        <label for="cfg-vol" style="display: inline-flex; align-items: center; gap: 6px">
          <HugeIcon name="volume-high" :size="16" />
          <span>Volume Bawaan: {{ settings.defaultVolume }}%</span>
        </label>
        <input
          id="cfg-vol"
          v-model.number="settings.defaultVolume"
          class="range-slider"
          type="range"
          min="0"
          max="100"
        />
      </div>

      <template v-if="platform?.supportsAudioDucking">
        <div class="switch-row">
          <div>
            <strong>Kecilkan Volume Aplikasi Lain Saat Pengumuman</strong>
            <p class="page-subtitle">
              Otomatis menurunkan volume musik atau aplikasi lain di Windows selama pengumuman
              diputar, lalu mengembalikannya setelah selesai
            </p>
          </div>
          <label class="toggle-switch">
            <input v-model="settings.duckOtherAudio" type="checkbox" />
            <span class="toggle-track"></span>
          </label>
        </div>

        <div v-if="settings.duckOtherAudio" class="field" style="padding: 10px 0 12px">
          <label for="cfg-duck-vol" style="display: inline-flex; align-items: center; gap: 6px">
            <HugeIcon name="volume-high" :size="16" />
            <span>Sisa Volume Aplikasi Lain: {{ settings.duckVolumePercent }}%</span>
          </label>
          <input
            id="cfg-duck-vol"
            v-model.number="settings.duckVolumePercent"
            class="range-slider"
            type="range"
            min="0"
            max="100"
          />
          <p class="page-subtitle" style="margin-top: 4px">
            Contoh: 25% menurunkan volume musik dari 80% menjadi 20% selama pengumuman berlangsung
          </p>
        </div>
      </template>

      <div v-if="platform?.supportsLiveAnnouncement" class="switch-row">
        <div>
          <strong>Tampilkan Tombol Pengumuman Langsung</strong>
          <p class="page-subtitle">
            Menampilkan tombol &ldquo;Pengumuman Langsung&rdquo; di Dasbor untuk berbicara langsung
            melalui mikrofon ke pengeras suara (disembunyikan secara bawaan)
          </p>
        </div>
        <label class="toggle-switch">
          <input v-model="settings.liveAnnouncementEnabled" type="checkbox" />
          <span class="toggle-track"></span>
        </label>
      </div>

      <div class="btn-row" style="margin-top: 16px">
        <button class="btn btn-primary" type="button" :disabled="busy" @click="emit('save-settings')">
          <HugeIcon name="checkmark-circle-02" :size="17" />
          <span>Simpan Pengaturan</span>
        </button>
      </div>
    </div>

    <div class="card">
      <h3 style="margin: 0 0 12px; font-size: 1rem">Log Aktivitas Terbaru</h3>
      <p v-if="logs.length === 0" class="page-subtitle">Belum ada catatan aktivitas.</p>
      <div v-else class="form-stack">
        <div v-for="(entry, idx) in logs.slice(0, 20)" :key="idx" class="log-item">
          <div style="display: flex; justify-content: space-between; gap: 8px; align-items: center">
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
          <p style="margin: 5px 0 0; font-size: 0.88rem">{{ entry.message }}</p>
        </div>
      </div>
    </div>
  </section>
</template>
