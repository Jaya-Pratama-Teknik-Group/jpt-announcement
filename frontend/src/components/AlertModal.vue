<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import HugeIcon from "./HugeIcon.vue";

const props = withDefaults(
  defineProps<{
    open: boolean;
    kind: "info" | "error";
    title?: string;
    message: string;
    nonce?: number;
    autoCloseSeconds?: number;
  }>(),
  {
    title: "",
    nonce: 0,
    autoCloseSeconds: 10
  }
);

const emit = defineEmits<{
  (e: "close"): void;
}>();

const totalSeconds = computed(() => Math.max(1, props.autoCloseSeconds));
const remainingSeconds = ref(totalSeconds.value);

const resolvedTitle = computed(() => {
  if (props.title) return props.title;
  return props.kind === "error" ? "Terjadi Kesalahan" : "Informasi";
});

const timerKey = computed(() => `${props.nonce}-${props.kind}-${props.message}`);

let closeTimeoutId: number | null = null;
let countdownIntervalId: number | null = null;

function clearTimers() {
  if (closeTimeoutId !== null) {
    window.clearTimeout(closeTimeoutId);
    closeTimeoutId = null;
  }
  if (countdownIntervalId !== null) {
    window.clearInterval(countdownIntervalId);
    countdownIntervalId = null;
  }
}

function startAutoCloseTimer() {
  clearTimers();
  const seconds = totalSeconds.value;
  remainingSeconds.value = seconds;

  countdownIntervalId = window.setInterval(() => {
    if (remainingSeconds.value > 1) {
      remainingSeconds.value -= 1;
    }
  }, 1000);

  closeTimeoutId = window.setTimeout(() => {
    clearTimers();
    emit("close");
  }, seconds * 1000);
}

watch(
  () => [props.open, props.nonce, props.message] as const,
  ([isOpen]) => {
    if (isOpen) {
      startAutoCloseTimer();
    } else {
      clearTimers();
    }
  },
  { immediate: true }
);

function onKeyDown(ev: KeyboardEvent) {
  if (!props.open) return;
  if (ev.key === "Escape" || ev.key === "Enter") {
    ev.preventDefault();
    emit("close");
  }
}

onMounted(() => {
  window.addEventListener("keydown", onKeyDown);
});

onUnmounted(() => {
  clearTimers();
  window.removeEventListener("keydown", onKeyDown);
});
</script>

<template>
  <Transition name="modal-fade">
    <div v-if="open" class="modal-backdrop modal-backdrop-alert" @click.self="emit('close')">
      <div
        class="modal-panel modal-confirm modal-alert"
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="alert-modal-title"
      >
      <div class="confirm-header">
        <div
          class="confirm-icon-ring"
          :class="kind === 'error' ? 'alert-icon-error' : 'alert-icon-info'"
        >
          <HugeIcon
            :name="kind === 'error' ? 'alert-02' : 'checkmark-circle-02'"
            :size="24"
          />
        </div>
        <button
          class="modal-close-btn"
          type="button"
          aria-label="Tutup notifikasi"
          @click="emit('close')"
        >
          <HugeIcon name="cancel-01" :size="18" />
        </button>
      </div>

      <div class="confirm-body">
        <h2 id="alert-modal-title" class="confirm-title">{{ resolvedTitle }}</h2>
        <div
          class="alert-message-box"
          :class="kind === 'error' ? 'alert-message-error' : 'alert-message-info'"
        >
          {{ message }}
        </div>

        <div class="alert-countdown-wrap">
          <div class="alert-countdown-row">
            <HugeIcon name="clock-01" :size="14" />
            <span>
              Menutup otomatis dalam <strong>{{ remainingSeconds }}</strong> detik
            </span>
          </div>
          <div class="alert-progress-track">
            <div
              :key="timerKey"
              class="alert-progress-fill"
              :class="kind === 'error' ? 'fill-error' : 'fill-info'"
              :style="{ animationDuration: `${totalSeconds}s` }"
            />
          </div>
        </div>
      </div>

      <div class="modal-footer modal-footer-alert">
        <button
          class="btn"
          :class="kind === 'error' ? 'btn-danger-solid' : 'btn-primary'"
          type="button"
          @click="emit('close')"
        >
          <HugeIcon name="checkmark-circle-02" :size="16" />
          <span>OK</span>
        </button>
      </div>
    </div>
  </div>
  </Transition>
</template>
