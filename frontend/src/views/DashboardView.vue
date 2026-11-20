<script setup lang="ts">
import HugeIcon from "../components/HugeIcon.vue";
import type { AppConfig, DashboardState } from "../types/models";

defineProps<{
  dashboard: DashboardState | null;
  settings: AppConfig;
  liveCountdownText: string;
}>();

const emit = defineEmits<{
  (e: "create-schedule"): void;
  (e: "toggle-live"): void;
  (e: "toggle-pause"): void;
  (e: "stop-audio"): void;
  (e: "play-schedule-now", scheduleId: string): void;
  (e: "navigate-schedules"): void;
}>();

function formatNextTime(iso: string): string {
  return new Date(iso).toLocaleTimeString("id-ID", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  });
}
</script>

<template>
  <section>
    <div class="page-header">
      <div>
        <h1 class="page-title">Dasbor Pengumuman</h1>
        <p class="page-subtitle">Pantau status penjadwal dan pemutaran audio terkini</p>
      </div>
      <div class="btn-row">
        <button
          v-if="
            dashboard?.platform.supportsLiveAnnouncement &&
            (settings.liveAnnouncementEnabled || dashboard?.playback.isLive)
          "
          class="btn"
          :class="dashboard?.playback.isLive ? 'btn-danger-solid' : 'btn-soft-accent'"
          type="button"
          @click="emit('toggle-live')"
        >
          <HugeIcon :name="dashboard?.playback.isLive ? 'stop' : 'mic-01'" :size="17" />
          <span>
            {{
              dashboard?.playback.isLive
                ? "Matikan Mikrofon (Aktif)"
                : "Pengumuman Langsung"
            }}
          </span>
        </button>

        <button class="btn btn-primary" type="button" @click="emit('create-schedule')">
          <HugeIcon name="add-01" :size="17" />
          <span>Tambah Jadwal</span>
        </button>

        <button
          v-if="dashboard?.playback.isPlaying && !dashboard.playback.isLive"
          class="btn btn-soft-primary"
          type="button"
          @click="emit('toggle-pause')"
        >
          <HugeIcon :name="dashboard.playback.isPaused ? 'play' : 'pause'" :size="17" />
          <span>{{ dashboard.playback.isPaused ? "Lanjutkan" : "Jeda" }}</span>
        </button>

        <button
          v-if="dashboard?.playback.isPlaying && !dashboard.playback.isLive"
          class="btn btn-soft-danger"
          type="button"
          @click="emit('stop-audio')"
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
          <span class="card-icon-chip chip-navy">
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
          <span class="card-icon-chip chip-coral">
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
        <p class="metric-value metric-value-sm">
          {{
            dashboard?.playback.isPlaying
              ? dashboard.playback.audioName || "Memutar..."
              : "Siaga"
          }}
        </p>
        <span
          class="badge"
          :class="
            dashboard?.playback.isPlaying
              ? dashboard.playback.isLive
                ? 'badge-danger'
                : dashboard.playback.isPaused
                  ? 'badge-warning'
                  : 'badge-accent'
              : 'badge-muted'
          "
        >
          {{
            dashboard?.playback.isPlaying
              ? dashboard.playback.isLive
                ? "Siaran Langsung"
                : dashboard.playback.isPaused
                  ? "Dijeda"
                  : "Sedang diputar"
              : "Tidak ada audio"
          }}
        </span>
      </div>
    </div>

    <div class="card hero-card">
      <div class="hero-layout">
        <div>
          <p class="hero-kicker">
            <HugeIcon name="notification-02" :size="14" />
            <span>Pengumuman Berikutnya</span>
          </p>

          <template v-if="dashboard?.nextAnnouncement">
            <h2 class="hero-title">{{ dashboard.nextAnnouncement.title }}</h2>
            <div class="hero-meta">
              <span class="badge badge-primary">
                Jam {{ formatNextTime(dashboard.nextAnnouncement.scheduledAt) }}
              </span>
              <span>Audio: {{ dashboard.nextAnnouncement.audioName || "-" }}</span>
              <span class="countdown-pill">
                <HugeIcon name="clock-01" :size="14" />
                <span>{{ liveCountdownText }}</span>
              </span>
            </div>
          </template>

          <template v-else>
            <h2 class="hero-title">Belum Ada Jadwal Aktif</h2>
            <p class="page-subtitle">
              Tambahkan berkas audio dan buat jadwal pertama Anda agar pemutaran berjalan otomatis.
            </p>
          </template>
        </div>

        <div class="btn-row">
          <template v-if="dashboard?.nextAnnouncement">
            <button
              class="btn btn-accent"
              type="button"
              @click="emit('play-schedule-now', dashboard.nextAnnouncement.scheduleId)"
            >
              <HugeIcon name="play" :size="17" />
              <span>Putar Sekarang</span>
            </button>
            <button
              v-if="dashboard.playback.isPlaying"
              class="btn btn-soft-danger"
              type="button"
              @click="emit('stop-audio')"
            >
              <HugeIcon name="stop" :size="17" />
              <span>Hentikan Audio</span>
            </button>
          </template>
          <button
            v-else
            class="btn btn-soft-primary"
            type="button"
            @click="emit('navigate-schedules')"
          >
            <HugeIcon name="calendar-03" :size="17" />
            <span>Kelola Jadwal</span>
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
