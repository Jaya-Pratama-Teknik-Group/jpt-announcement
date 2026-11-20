<script setup lang="ts">
import { onMounted, onUnmounted } from "vue";
import HugeIcon from "./HugeIcon.vue";

const props = defineProps<{
  open: boolean;
  title: string;
  itemName: string;
  itemDetail?: string;
  description: string;
  busy?: boolean;
}>();

const emit = defineEmits<{
  (e: "close"): void;
  (e: "confirm"): void;
}>();

function onKeyDown(ev: KeyboardEvent) {
  if (ev.key === "Escape" && props.open && !props.busy) {
    emit("close");
  }
}

onMounted(() => {
  window.addEventListener("keydown", onKeyDown);
});

onUnmounted(() => {
  window.removeEventListener("keydown", onKeyDown);
});
</script>

<template>
  <Transition name="modal-fade">
    <div v-if="open" class="modal-backdrop" @click.self="!busy && emit('close')">
      <div
        class="modal-panel modal-confirm"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="confirm-delete-title"
      >
      <div class="confirm-header">
        <div class="confirm-icon-ring">
          <HugeIcon name="delete-02" :size="24" />
        </div>
        <button
          class="modal-close-btn"
          type="button"
          aria-label="Tutup dialog"
          :disabled="busy"
          @click="emit('close')"
        >
          <HugeIcon name="cancel-01" :size="18" />
        </button>
      </div>

      <div class="confirm-body">
        <h2 id="confirm-delete-title" class="confirm-title">{{ title }}</h2>
        <p class="confirm-desc">{{ description }}</p>

        <div class="confirm-target-box">
          <strong>{{ itemName }}</strong>
          <span v-if="itemDetail" class="page-subtitle">{{ itemDetail }}</span>
        </div>
      </div>

      <div class="modal-footer modal-footer-confirm">
        <button class="btn" type="button" :disabled="busy" @click="emit('close')">Batal</button>
        <button
          class="btn btn-danger-solid"
          type="button"
          :disabled="busy"
          @click="emit('confirm')"
        >
          <HugeIcon name="delete-02" :size="16" />
          <span>{{ busy ? "Menghapus..." : "Ya, Hapus" }}</span>
        </button>
      </div>
    </div>
  </div>
  </Transition>
</template>
