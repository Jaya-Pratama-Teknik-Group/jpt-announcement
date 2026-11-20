<script setup lang="ts">
import { computed } from "vue";
import HugeIcon from "../components/HugeIcon.vue";
import { formatTimes } from "../utils/formatters";
import type { AppConfig, DashboardState, Schedule } from "../types/models";

interface TodayAnnouncementSlot {
  key: string;
  scheduleId: string;
  title: string;
  time: string;
  slotMinutes: number;
  audioName: string;
  volume: number;
  status: "done" | "next" | "upcoming";
}

const props = defineProps<{
  dashboard: DashboardState | null;
  schedules: Schedule[];
  audioNameMap: Map<string, string>;
  settings: AppConfig;
  countdownTick: number;
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

function isSameLocalMinute(iso: string | undefined, targetDate: Date, hh: number, mm: number): boolean {
  if (!iso) return false;
  const d = new Date(iso);
  return (
    d.getFullYear() === targetDate.getFullYear() &&
    d.getMonth() === targetDate.getMonth() &&
    d.getDate() === targetDate.getDate() &&
    d.getHours() === hh &&
    d.getMinutes() === mm
  );
}

const todayLabel = computed(() =>
  new Date(props.countdownTick).toLocaleDateString("id-ID", {
    weekday: "long",
    day: "numeric",
    month: "short"
  })
);

const todaySlots = computed<TodayAnnouncementSlot[]>(() => {
  const now = new Date(props.countdownTick);
  const weekday = now.getDay();
  const nowMinutes = now.getHours() * 60 + now.getMinutes();

  const next = props.dashboard?.nextAnnouncement;
  const nextDate = next ? new Date(next.scheduledAt) : null;
  const isNextToday =
    nextDate !== null &&
    nextDate.getFullYear() === now.getFullYear() &&
    nextDate.getMonth() === now.getMonth() &&
    nextDate.getDate() === now.getDate();
  const nextTimeHHMM =
    isNextToday && nextDate
      ? `${String(nextDate.getHours()).padStart(2, "0")}:${String(nextDate.getMinutes()).padStart(2, "0")}`
      : "";

  const slots: TodayAnnouncementSlot[] = [];

  for (const sch of props.schedules) {
    if (!sch.enabled || !sch.audioId) continue;
    const audioName = props.audioNameMap.get(sch.audioId);
    if (!audioName) continue;
    if (sch.days.length > 0 && !sch.days.includes(weekday)) continue;

    for (const timeStr of formatTimes(sch)) {
      const [hhStr, mmStr] = timeStr.split(":");
      const hh = Number(hhStr);
      const mm = Number(mmStr);
      if (!Number.isInteger(hh) || !Number.isInteger(mm)) continue;

      const slotMinutes = hh * 60 + mm;
      const isNext =
        isNextToday && next?.scheduleId === sch.id && nextTimeHHMM === timeStr;
      const triggeredThisMinute = isSameLocalMinute(sch.lastTriggeredAt, now, hh, mm);

      let status: TodayAnnouncementSlot["status"] = "upcoming";
      if (isNext) {
        status = "next";
      } else if (slotMinutes < nowMinutes || (slotMinutes === nowMinutes && triggeredThisMinute)) {
        status = "done";
      }

      slots.push({
        key: `${sch.id}:${timeStr}`,
        scheduleId: sch.id,
        title: sch.title,
        time: timeStr,
        slotMinutes,
        audioName,
        volume: sch.volume,
        status
      });
    }
  }

  slots.sort((a, b) => a.slotMinutes - b.slotMinutes || a.title.localeCompare(b.title, "id"));
  return slots;
});

const remainingTodayCount = computed(
  () => todaySlots.value.filter((slot) => slot.status !== "done").length
);
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

    <div class="card hero-card" style="margin-bottom: 16px">
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

    <div class="card">
      <div class="today-card-header">
        <div>
          <h3 class="today-card-title">Jadwal Hari Ini</h3>
          <p class="page-subtitle">{{ todayLabel }}</p>
        </div>
        <span
          class="badge"
          :class="remainingTodayCount > 0 ? 'badge-primary' : 'badge-muted'"
        >
          <template v-if="todaySlots.length === 0">0 jadwal</template>
          <template v-else-if="remainingTodayCount === 0">
            Semua selesai ({{ todaySlots.length }})
          </template>
          <template v-else>
            {{ remainingTodayCount }} tersisa &bull; {{ todaySlots.length }} total
          </template>
        </span>
      </div>

      <div v-if="todaySlots.length === 0" class="today-empty">
        <HugeIcon name="calendar-03" :size="18" />
        <span>Tidak ada pengumuman yang dijadwalkan berbunyi hari ini.</span>
      </div>

      <div v-else class="today-list">
        <div
          v-for="slot in todaySlots"
          :key="slot.key"
          class="today-row"
          :class="{ 'is-done': slot.status === 'done', 'is-next': slot.status === 'next' }"
        >
          <div class="today-row-left">
            <span class="time-pill" :class="{ 'time-pill-accent': slot.status === 'next' }">
              {{ slot.time }}
            </span>
            <div class="today-row-info">
              <div class="today-row-title-line">
                <strong class="today-row-title">{{ slot.title }}</strong>
                <span
                  class="badge"
                  :class="
                    slot.status === 'next'
                      ? 'badge-accent'
                      : slot.status === 'done'
                        ? 'badge-muted'
                        : 'badge-primary'
                  "
                >
                  {{
                    slot.status === "next"
                      ? "Berikutnya"
                      : slot.status === "done"
                        ? "Selesai"
                        : "Akan datang"
                  }}
                </span>
              </div>
              <p class="page-subtitle today-row-meta">
                {{ slot.audioName }} &bull; Volume {{ slot.volume }}%
              </p>
            </div>
          </div>

          <button
            class="btn btn-sm"
            :class="slot.status === 'next' ? 'btn-soft-accent' : 'btn-soft-primary'"
            type="button"
            @click="emit('play-schedule-now', slot.scheduleId)"
          >
            <HugeIcon name="play" :size="14" />
            <span>Putar</span>
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
