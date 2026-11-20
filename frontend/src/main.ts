import { createApp } from "vue";
import App from "./App.vue";
import { registerHugeIcons } from "./icons/register";
import { initTheme } from "./utils/theme";
import "./styles/main.css";

initTheme();
registerHugeIcons();
createApp(App).mount("#app");
