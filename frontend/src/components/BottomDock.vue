<script setup lang="ts">
import HugeIcon from "./HugeIcon.vue";
import type { HugeIconName } from "../icons/register";
import type { NavTab } from "../types/models";

export interface DockNavItem {
  id: NavTab;
  label: string;
  short: string;
  icon: HugeIconName;
}

defineProps<{
  items: DockNavItem[];
  activeTab: NavTab;
}>();

const emit = defineEmits<{
  (e: "select", tab: NavTab): void;
}>();
</script>

<template>
  <div class="dock-wrap">
    <nav class="bottom-dock" aria-label="Navigasi utama">
      <button
        v-for="item in items"
        :key="item.id"
        type="button"
        class="dock-item"
        :class="{ active: activeTab === item.id }"
        :aria-current="activeTab === item.id ? 'page' : undefined"
        @click="emit('select', item.id)"
      >
        <span class="dock-item-icon">
          <HugeIcon :name="item.icon" :size="19" />
        </span>
        <span class="dock-item-label dock-label-short">{{ item.short }}</span>
        <span class="dock-item-label dock-label-full">{{ item.label }}</span>
        <span class="dock-item-dot" aria-hidden="true"></span>
      </button>
    </nav>
  </div>
</template>
