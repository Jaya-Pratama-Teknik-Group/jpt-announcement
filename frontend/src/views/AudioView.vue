<script setup lang="ts">
import { ref } from "vue";
import HugeIcon from "../components/HugeIcon.vue";
import { formatSize } from "../utils/formatters";
import type { AudioAsset, PlaybackState } from "../types/models";

const props = defineProps<{
  audios: AudioAsset[];
  playback?: PlaybackState;
  supportsNativeFilePicker: boolean;
  busy: boolean;
}>();

const emit = defineEmits<{
  (e: "pick-audio"): void;
  (e: "import-audio-file", file: File): void;
  (e: "play-audio", audioId: string): void;
  (e: "stop-audio"): void;
  (e: "delete-audio", asset: AudioAsset): void;
}>();

const fileInputRef = ref<HTMLInputElement | null>(null);

function onPickClick() {
  if (props.supportsNativeFilePicker) {
    emit("pick-audio");
    return;
  }
  fileInputRef.value?.click();
}

function onFileSelected(ev: Event) {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (file) {
    emit("import-audio-file", file);
  }
}
</script>

<template>
  <section>
    <div class="page-header">
      <div>
        <h1 class="page-title">Manajemen Audio</h1>
        <p class="page-subtitle">Unggah dan uji berkas audio lokal (.mp3, .mpeg, .mpga, atau .wav)</p>
      </div>
      <div class="btn-row">
        <button class="btn btn-primary" type="button" :disabled="busy" @click="onPickClick">
          <HugeIcon name="add-01" :size="17" />
          <span>Tambah Audio</span>
        </button>
        <button
          v-if="playback?.isPlaying"
          class="btn btn-soft-danger"
          type="button"
          @click="emit('stop-audio')"
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
        @change="onFileSelected"
      />
    </div>

    <div v-if="audios.length === 0" class="card empty-state">
      <span class="empty-state-icon">
        <HugeIcon name="music-note-03" :size="22" />
      </span>
      <p class="page-subtitle" style="margin: 0">
        Belum ada berkas audio. Klik <strong>Tambah Audio</strong> untuk memilih berkas .mp3, .mpeg,
        atau .wav dari perangkat Anda.
      </p>
    </div>

    <div v-else class="card list-card">
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
            v-if="playback?.isPlaying && playback.audioId === asset.id"
            class="btn btn-sm btn-soft-danger"
            type="button"
            @click="emit('stop-audio')"
          >
            <HugeIcon name="stop" :size="15" />
            <span>Hentikan</span>
          </button>
          <button
            v-else
            class="btn btn-sm btn-soft-primary"
            type="button"
            @click="emit('play-audio', asset.id)"
          >
            <HugeIcon name="play" :size="15" />
            <span>Uji Putar</span>
          </button>
          <button
            class="btn btn-sm btn-soft-danger"
            type="button"
            @click="emit('delete-audio', asset)"
          >
            <HugeIcon name="delete-02" :size="15" />
            <span>Hapus</span>
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
