<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue";
import HugeIcon from "./HugeIcon.vue";
import { HARI_INDONESIA, type AudioAsset, type PlaybackState, type Schedule } from "../types/models";

const props = defineProps<{
  open: boolean;
  initial: Schedule | null;
  audios: AudioAsset[];
  defaultVolume: number;
  playback?: PlaybackState;
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "save", item: Schedule): void;
  (e: "preview-audio", audioId: string, volume: number): void;
  (e: "stop-audio"): void;
}>();

const hourDraft = ref("07");
const minuteDraft = ref("00");

const hours24 = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, "0"));
const minutes60 = Array.from({ length: 60 }, (_, i) => String(i).padStart(2, "0"));

function setDraftFromTime(hhmm: string) {
  const [h = "07", m = "00"] = hhmm.split(":");
  hourDraft.value = hours24.includes(h) ? h : "07";
  minuteDraft.value = minutes60.includes(m) ? m : "00";
}

function currentDraftTime(): string {
  return `${hourDraft.value}:${minuteDraft.value}`;
}

const form = reactive<Schedule>({
  id: "",
  title: "",
  description: "",
  audioId: "",
  timeOfDay: "07:00",
  timesOfDay: ["07:00"],
  days: [],
  enabled: true,
  volume: 85
});

function normalizeTimesList(list: string[]): string[] {
  const unique = Array.from(new Set(list.map((t) => t.trim()).filter(Boolean)));
  unique.sort();
  return unique;
}

watch(
  () => [props.open, props.initial] as const,
  ([isOpen, item]) => {
    if (!isOpen) return;
    if (item) {
      const initialTimes =
        item.timesOfDay && item.timesOfDay.length > 0
          ? item.timesOfDay
          : item.timeOfDay
            ? [item.timeOfDay]
            : ["07:00"];
      const normalized = normalizeTimesList(initialTimes);
      form.id = item.id;
      form.title = item.title;
      form.description = item.description;
      form.audioId = item.audioId;
      form.timesOfDay = normalized;
      form.timeOfDay = normalized[0] ?? "07:00";
      setDraftFromTime(form.timeOfDay);
      form.days = [...(item.days ?? [])];
      form.enabled = item.enabled;
      form.volume = item.volume;
    } else {
      form.id = "";
      form.title = "";
      form.description = "";
      form.audioId = props.audios[0]?.id ?? "";
      form.timeOfDay = "07:00";
      form.timesOfDay = ["07:00"];
      setDraftFromTime("07:00");
      form.days = [1, 2, 3, 4, 5];
      form.enabled = true;
      form.volume = props.defaultVolume;
    }
  },
  { immediate: true }
);

function addTime() {
  const value = currentDraftTime();
  form.timesOfDay = normalizeTimesList([...form.timesOfDay, value]);
  form.timeOfDay = form.timesOfDay[0] ?? value;
}

function removeTime(target: string) {
  form.timesOfDay = form.timesOfDay.filter((t) => t !== target);
  form.timeOfDay = form.timesOfDay[0] ?? "";
}

function toggleDay(dayValue: number) {
  const idx = form.days.indexOf(dayValue);
  if (idx >= 0) {
    form.days.splice(idx, 1);
  } else {
    form.days.push(dayValue);
  }
}

const isPreviewingSelected = computed(
  () => Boolean(props.playback?.isPlaying && props.playback.audioId === form.audioId)
);

watch(
  () => form.audioId,
  (nextAudioID) => {
    if (!nextAudioID) {
      form.enabled = false;
    }
  }
);

function togglePreviewAudio() {
  if (!form.audioId) return;
  if (isPreviewingSelected.value) {
    emit("stop-audio");
    return;
  }
  emit("preview-audio", form.audioId, Number(form.volume));
}

function onKeyDown(ev: KeyboardEvent) {
  if (ev.key === "Escape" && props.open) {
    emit("close");
  }
}

onMounted(() => {
  window.addEventListener("keydown", onKeyDown);
});

onUnmounted(() => {
  window.removeEventListener("keydown", onKeyDown);
});

function submitForm() {
  let times = normalizeTimesList(form.timesOfDay);
  if (times.length === 0) {
    times = [currentDraftTime()];
  }

  emit("save", {
    id: form.id,
    title: form.title.trim(),
    description: form.description.trim(),
    audioId: form.audioId,
    timeOfDay: times[0],
    timesOfDay: times,
    days: [...form.days],
    enabled: form.enabled,
    volume: Number(form.volume)
  });
}
</script>

<template>
  <Transition name="modal-fade">
    <div v-if="open" class="modal-backdrop" @click.self="emit('close')">
      <div class="modal-panel" role="dialog" aria-modal="true" aria-labelledby="schedule-modal-title">
      <div class="modal-header">
        <div class="modal-title-group">
          <div class="modal-title-icon">
            <HugeIcon name="calendar-03" :size="20" />
          </div>
          <div>
            <h2 id="schedule-modal-title">
              {{ form.id ? "Ubah Jadwal Pengumuman" : "Tambah Jadwal Pengumuman" }}
            </h2>
            <p class="modal-subtitle">
              Atur jam 24-jam, hari aktif, dan audio pengumuman
            </p>
          </div>
        </div>
        <button
          class="modal-close-btn"
          type="button"
          aria-label="Tutup form jadwal"
          @click="emit('close')"
        >
          <HugeIcon name="cancel-01" :size="18" />
        </button>
      </div>

      <form class="modal-body" @submit.prevent="submitForm">
        <div class="modal-section">
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
            <label for="sch-desc">Catatan (Opsional)</label>
            <textarea
              id="sch-desc"
              v-model="form.description"
              class="textarea"
              rows="2"
              placeholder="Keterangan tambahan jadwal"
            ></textarea>
          </div>
        </div>

        <div class="modal-section">
          <div class="field">
            <label for="sch-hour">Jam Putar (Format 24 Jam — bisa lebih dari satu)</label>
            <div class="time-input-row">
              <div class="time-24h-picker">
                <select
                  id="sch-hour"
                  v-model="hourDraft"
                  class="select time-select"
                  aria-label="Jam (00 sampai 23)"
                >
                  <option v-for="h in hours24" :key="h" :value="h">{{ h }}</option>
                </select>
                <span class="time-colon">:</span>
                <select
                  id="sch-minute"
                  v-model="minuteDraft"
                  class="select time-select"
                  aria-label="Menit (00 sampai 59)"
                >
                  <option v-for="m in minutes60" :key="m" :value="m">{{ m }}</option>
                </select>
              </div>
              <button class="btn btn-sm" type="button" @click="addTime">
                <HugeIcon name="add-01" :size="15" />
                <span>Tambah Jam</span>
              </button>
            </div>
            <div v-if="form.timesOfDay.length > 0" class="time-pills" style="margin-top: 6px">
              <span v-for="t in form.timesOfDay" :key="t" class="time-chip">
                <span>{{ t }}</span>
                <button
                  type="button"
                  class="time-chip-remove"
                  :aria-label="`Hapus jam ${t}`"
                  @click="removeTime(t)"
                >
                  &times;
                </button>
              </span>
            </div>
            <p v-else class="page-subtitle" style="margin: 2px 0 0">
              Pilih jam lalu klik <strong>Tambah Jam</strong> (minimal 1 jam).
            </p>
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
        </div>

        <div class="modal-section">
          <div class="field">
            <label for="sch-audio">Berkas Audio</label>
            <div class="audio-select-row">
              <select id="sch-audio" v-model="form.audioId" class="select">
                <option value="" disabled>-- Pilih berkas audio (.mp3 / .mpeg / .wav) --</option>
                <option v-for="audio in audios" :key="audio.id" :value="audio.id">
                  {{ audio.name }} (.{{ audio.format }})
                </option>
              </select>
              <button
                class="btn btn-sm"
                :class="isPreviewingSelected ? 'btn-danger' : ''"
                type="button"
                :disabled="!form.audioId"
                @click="togglePreviewAudio"
              >
                <HugeIcon :name="isPreviewingSelected ? 'stop' : 'play'" :size="15" />
                <span>{{ isPreviewingSelected ? "Hentikan" : "Putar" }}</span>
              </button>
            </div>
            <p v-if="!form.audioId" class="page-subtitle" style="margin: 2px 0 0; color: var(--warning)">
              Pilih berkas audio terlebih dahulu agar jadwal dapat diaktifkan.
            </p>
          </div>

          <div class="field">
            <label for="sch-volume" style="display: inline-flex; align-items: center; gap: 6px">
              <HugeIcon name="volume-high" :size="16" />
              <span>Volume: {{ form.volume }}%</span>
            </label>
            <input
              id="sch-volume"
              v-model.number="form.volume"
              class="range-slider"
              type="range"
              min="0"
              max="100"
            />
          </div>

          <div class="switch-row" style="padding: 4px 0 0">
            <div>
              <strong>Jadwal Aktif</strong>
              <p class="page-subtitle">
                {{
                  form.audioId
                    ? "Putar otomatis saat waktu jadwal tercapai"
                    : "Pilih audio terlebih dahulu sebelum mengaktifkan jadwal"
                }}
              </p>
            </div>
            <label class="toggle-switch">
              <input v-model="form.enabled" type="checkbox" :disabled="!form.audioId" />
              <span class="toggle-track"></span>
            </label>
          </div>
        </div>

        <div class="modal-footer" style="margin: 4px -20px -18px">
          <button class="btn" type="button" @click="emit('close')">Batal</button>
          <button
            class="btn btn-primary"
            type="submit"
            :disabled="form.enabled && !form.audioId"
          >
            <HugeIcon name="checkmark-circle-02" :size="17" />
            <span>Simpan Jadwal</span>
          </button>
        </div>
      </form>
    </div>
  </div>
  </Transition>
</template>
