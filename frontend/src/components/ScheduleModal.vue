<script setup lang="ts">
import { reactive, watch } from "vue";
import HugeIcon from "./HugeIcon.vue";
import { HARI_INDONESIA, type AudioAsset, type Schedule } from "../types/models";

const props = defineProps<{
  open: boolean;
  initial: Schedule | null;
  audios: AudioAsset[];
  defaultVolume: number;
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "save", item: Schedule): void;
}>();

const form = reactive<Schedule>({
  id: "",
  title: "",
  description: "",
  audioId: "",
  timeOfDay: "07:00",
  days: [],
  enabled: true,
  volume: 85
});

watch(
  () => [props.open, props.initial] as const,
  ([isOpen, item]) => {
    if (!isOpen) return;
    if (item) {
      form.id = item.id;
      form.title = item.title;
      form.description = item.description;
      form.audioId = item.audioId;
      form.timeOfDay = item.timeOfDay;
      form.days = [...(item.days ?? [])];
      form.enabled = item.enabled;
      form.volume = item.volume;
    } else {
      form.id = "";
      form.title = "";
      form.description = "";
      form.audioId = props.audios[0]?.id ?? "";
      form.timeOfDay = "07:00";
      form.days = [1, 2, 3, 4, 5];
      form.enabled = true;
      form.volume = props.defaultVolume;
    }
  },
  { immediate: true }
);

function toggleDay(dayValue: number) {
  const idx = form.days.indexOf(dayValue);
  if (idx >= 0) {
    form.days.splice(idx, 1);
  } else {
    form.days.push(dayValue);
  }
}

function submitForm() {
  emit("save", {
    id: form.id,
    title: form.title.trim(),
    description: form.description.trim(),
    audioId: form.audioId,
    timeOfDay: form.timeOfDay,
    days: [...form.days],
    enabled: form.enabled,
    volume: Number(form.volume)
  });
}
</script>

<template>
  <div v-if="open" class="modal-backdrop" @click.self="emit('close')">
    <div class="modal-panel" role="dialog" aria-modal="true">
      <div class="modal-header">
        <h2 style="display: inline-flex; align-items: center; gap: 8px">
          <HugeIcon name="calendar-03" :size="20" />
          <span>{{ form.id ? "Ubah Jadwal Pengumuman" : "Tambah Jadwal Pengumuman" }}</span>
        </h2>
        <button class="btn btn-sm" type="button" @click="emit('close')">
          <HugeIcon name="cancel-01" :size="16" />
          <span>Tutup</span>
        </button>
      </div>

      <form class="form-stack" @submit.prevent="submitForm">
        <div class="field">
          <label for="sch-title">Judul Pengumuman</label>
          <input
            id="sch-title"
            v-model="form.title"
            class="input"
            type="text"
            placeholder="Contoh: Bel Masuk Pagi"
            required
          />
        </div>

        <div class="field">
          <label for="sch-time">Jam Putar (24 Jam)</label>
          <input id="sch-time" v-model="form.timeOfDay" class="input" type="time" required />
        </div>

        <div class="field">
          <label for="sch-audio">Berkas Audio</label>
          <select id="sch-audio" v-model="form.audioId" class="select" required>
            <option value="" disabled>-- Pilih berkas audio (.mp3 / .mpeg / .wav) --</option>
            <option v-for="audio in audios" :key="audio.id" :value="audio.id">
              {{ audio.name }} (.{{ audio.format }})
            </option>
          </select>
        </div>

        <div class="field">
          <label>Hari Aktif (kosongkan = Setiap Hari)</label>
          <div class="day-chips">
            <button
              v-for="hari in HARI_INDONESIA"
              :key="hari.value"
              type="button"
              class="day-chip"
              :class="{ selected: form.days.includes(hari.value) }"
              @click="toggleDay(hari.value)"
            >
              {{ hari.short }}
            </button>
          </div>
        </div>

        <div class="field">
          <label for="sch-volume">Volume: {{ form.volume }}%</label>
          <input id="sch-volume" v-model.number="form.volume" type="range" min="0" max="100" />
        </div>

        <div class="field">
          <label for="sch-desc">Catatan (Opsional)</label>
          <textarea
            id="sch-desc"
            v-model="form.description"
            class="textarea"
            rows="2"
            placeholder="Keterangan tambahan jadwal"
          ></textarea>
        </div>

        <div class="switch-row">
          <div>
            <strong>Jadwal Aktif</strong>
            <p class="page-subtitle">Putar otomatis saat waktu jadwal tercapai</p>
          </div>
          <input v-model="form.enabled" class="switch-input" type="checkbox" />
        </div>

        <div class="btn-row">
          <button class="btn btn-primary" type="submit" :disabled="!form.audioId">
            <HugeIcon name="checkmark-circle-02" :size="17" />
            <span>Simpan Jadwal</span>
          </button>
          <button class="btn" type="button" @click="emit('close')">Batal</button>
        </div>
      </form>
    </div>
  </div>
</template>
