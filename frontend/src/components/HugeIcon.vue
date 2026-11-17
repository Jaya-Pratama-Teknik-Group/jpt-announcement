<script setup lang="ts">
import { Icon } from "@iconify/vue";
import type { HugeIconName } from "../icons/register";

const props = withDefaults(
  defineProps<{
    name: HugeIconName;
    size?: number;
  }>(),
  {
    size: 18
  }
);
</script>

<template>
  <Icon
    :icon="`hugeicons:${props.name}`"
    :width="props.size"
    :height="props.size"
    aria-hidden="true"
  />
</template>
