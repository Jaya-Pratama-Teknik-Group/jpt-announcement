<script setup lang="ts">
import { ref } from "vue";
import HugeIcon from "../components/HugeIcon.vue";
import { formatDays, formatTimes } from "../utils/formatters";
import type { AudioAsset, Schedule } from "../types/models";

const props = defineProps<{
  schedules: Schedule[];
  audios: AudioAsset[];
  audioNameMap: Map<string, string>;
  schedulesWithoutAudioCount: number;
  supportsNativeFilePicker: boolean;
  busy: boolean;
}>();

const emit = defineEmits<{
  (e: "export-schedules"): void;
  (e: "import-schedules"): void;
  (e: "import-schedule-file", file: File): void;
  (e: "create-schedule"): void;
  (e: "play-schedule-now", scheduleId: string): void;
  (e: "toggle-schedule", item: Schedule): void;
  (e: "edit-schedule", item: Schedule): void;
  (e: "delete-schedule", item: Schedule): void;
}>();

const scheduleImportInputRef = ref<HTMLInputElement | null>(null);

function hasValidAudio(item: Schedule): boolean {
  return Boolean(item.audioId && props.audioNameMap.has(item.audioId));
}

function onImportClick() {
  if (props.supportsNativeFilePicker) {
    emit("import-schedules");
    return;
  }
  scheduleImportInputRef.value?.click();
}

function onFileSelected(ev: Event) {
  const input = ev.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (file) {
    emit("import-schedule-file", file);
  }
}
</script>

<template>
  <section>
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
          @click="emit('export-schedules')"
        >
          <HugeIcon name="file-export" :size="17" />
          <span>Ekspor</span>
        </button>
        <button class="btn" type="button" :disabled="busy" @click="onImportClick">
          <HugeIcon name="file-import" :size="17" />
          <span>Impor</span>
        </button>
        <button
          class="btn btn-primary"
          type="button"
          :disabled="audios.length === 0"
          @click="emit('create-schedule')"
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
        @change="onFileSelected"
      />
    </div>

    <div v-if="audios.length === 0" class="notice notice-warning">
      Tambahkan minimal satu berkas audio di menu <strong>Audio</strong> sebelum membuat atau
      mengaktifkan jadwal.
    </div>

    <div v-if="schedulesWithoutAudioCount > 0" class="notice notice-warning">
      Ada <strong>{{ schedulesWithoutAudioCount }} jadwal</strong> yang belum memiliki berkas audio
      (hasil impor). Klik <strong>Pilih Audio</strong> pada jadwal tersebut agar dapat diaktifkan.
    </div>

    <div v-if="schedules.length === 0" class="card empty-state">
      <span class="empty-state-icon">
        <HugeIcon name="calendar-03" :size="22" />
      </span>
      <p class="page-subtitle" style="margin: 0">Belum ada jadwal pengumuman yang tersimpan.</p>
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
                  class="btn btn-sm btn-soft-primary"
                  type="button"
                  :disabled="!hasValidAudio(item)"
                  @click="emit('play-schedule-now', item.id)"
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
                  @click="emit('toggle-schedule', item)"
                >
                  <HugeIcon name="power" :size="15" />
                  <span>{{ item.enabled ? "Matikan" : "Aktifkan" }}</span>
                </button>
                <button
                  class="btn btn-sm"
                  :class="!hasValidAudio(item) ? 'btn-accent' : ''"
                  type="button"
                  @click="emit('edit-schedule', item)"
                >
                  <HugeIcon name="edit-02" :size="15" />
                  <span>{{ hasValidAudio(item) ? "Ubah" : "Pilih Audio" }}</span>
                </button>
                <button
                  class="btn btn-sm btn-soft-danger"
                  type="button"
                  @click="emit('delete-schedule', item)"
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
            <h3 style="margin: 0; font-size: 1rem">{{ item.title }}</h3>
            <div class="time-pills" style="margin: 8px 0 6px">
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
            @click="emit('play-schedule-now', item.id)"
          >
            <HugeIcon name="play" :size="15" />
            <span>Putar</span>
          </button>
          <button
            class="btn btn-sm"
            type="button"
            :disabled="!item.enabled && !hasValidAudio(item)"
            @click="emit('toggle-schedule', item)"
          >
            <HugeIcon name="power" :size="15" />
            <span>{{ item.enabled ? "Nonaktifkan" : "Aktifkan" }}</span>
          </button>
          <button
            class="btn btn-sm"
            :class="!hasValidAudio(item) ? 'btn-accent' : ''"
            type="button"
            @click="emit('edit-schedule', item)"
          >
            <HugeIcon name="edit-02" :size="15" />
            <span>{{ hasValidAudio(item) ? "Ubah" : "Pilih Audio" }}</span>
          </button>
          <button
            class="btn btn-sm btn-soft-danger"
            type="button"
            @click="emit('delete-schedule', item)"
          >
            <HugeIcon name="delete-02" :size="15" />
            <span>Hapus</span>
          </button>
        </div>
      </div>
    </div>
  </section>
</template>
