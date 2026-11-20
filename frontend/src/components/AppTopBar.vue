<script setup lang="ts">
import logoImg from "../../logo.jpg";
import HugeIcon from "./HugeIcon.vue";
import type { PlaybackState } from "../types/models";

defineProps<{
  schedulerRunning: boolean;
  platformName?: string;
  playback?: PlaybackState;
}>();
</script>

<template>
  <header class="top-bar">
    <div class="top-bar-inner">
      <div class="brand">
        <div class="brand-logo-wrap">
          <img :src="logoImg" alt="Logo JPT Pengumuman Terjadwal" class="brand-logo" />
        </div>
        <div class="brand-text">
          <p class="brand-title">JPT Announce</p>
          <p class="brand-sub">
            {{ platformName === "android" ? "Mode Android" : "Pengumuman Terjadwal" }}
          </p>
        </div>
      </div>

      <div class="top-bar-status">
        <span
          v-if="playback?.isPlaying"
          class="badge"
          :class="playback.isLive ? 'badge-danger' : playback.isPaused ? 'badge-warning' : 'badge-accent'"
        >
          <HugeIcon :name="playback.isLive ? 'mic-01' : 'audio-wave-01'" :size="14" />
          <span>
            {{
              playback.isLive
                ? "Siaran Langsung"
                : playback.isPaused
                  ? "Dijeda"
                  : "Memutar"
            }}
          </span>
        </span>

        <span
          class="badge"
          :class="schedulerRunning ? 'badge-success' : 'badge-muted'"
          :title="schedulerRunning ? 'Penjadwal aktif' : 'Penjadwal nonaktif'"
        >
          <span class="status-dot" :class="{ pulsing: schedulerRunning }"></span>
          <span>{{ schedulerRunning ? "Penjadwal Aktif" : "Nonaktif" }}</span>
        </span>
      </div>
    </div>
  </header>
</template>
