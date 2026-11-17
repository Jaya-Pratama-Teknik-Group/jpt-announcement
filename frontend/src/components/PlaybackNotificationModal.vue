<script setup lang="ts">
import HugeIcon from "./HugeIcon.vue";
import type { PlaybackState } from "../types/models";

defineProps<{
  open: boolean;
  playback: PlaybackState | null | undefined;
}>();

const emit = defineEmits<{
  (e: "toggle-pause"): void;
  (e: "stop"): void;
  (e: "dismiss"): void;
}>();
</script>

<template>
  <Transition name="toast-slide">
    <aside
      v-if="open && playback?.isPlaying"
      class="playback-notify-card"
      role="status"
      aria-live="polite"
    >
      <div class="playback-notify-head">
        <div class="playback-notify-badge" :class="{ paused: playback.isPaused }">
          <HugeIcon
            :name="playback.isPaused ? 'pause' : 'notification-02'"
            :size="18"
          />
        </div>
        <div class="playback-notify-meta">
          <p class="playback-notify-kicker">
            {{ playback.isPaused ? "Pengumuman Dijeda" : "Pengumuman Sedang Diputar" }}
          </p>
          <h4 class="playback-notify-title">
            {{ playback.scheduleTitle || "Jadwal Pengumuman" }}
          </h4>
          <p class="playback-notify-audio">
            <HugeIcon name="music-note-03" :size="13" />
            <span>{{ playback.audioName || "Audio Pengumuman" }}</span>
          </p>
        </div>
      </div>

      <div class="playback-notify-actions">
        <button
          type="button"
          class="btn btn-soft-primary btn-sm"
          @click="emit('toggle-pause')"
        >
          <HugeIcon :name="playback.isPaused ? 'play' : 'pause'" :size="15" />
          <span>{{ playback.isPaused ? "Lanjutkan" : "Jeda Audio" }}</span>
        </button>
        <button
          type="button"
          class="btn btn-soft-danger btn-sm"
          @click="emit('stop')"
        >
          <HugeIcon name="stop" :size="15" />
          <span>Hentikan</span>
        </button>
        <button
          type="button"
          class="btn btn-primary btn-sm"
          @click="emit('dismiss')"
        >
          <HugeIcon name="checkmark-circle-02" :size="15" />
          <span>OK</span>
        </button>
      </div>
    </aside>
  </Transition>
</template>
