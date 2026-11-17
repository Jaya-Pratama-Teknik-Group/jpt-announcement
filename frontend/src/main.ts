import { createApp } from "vue";
import App from "./App.vue";
import { registerHugeIcons } from "./icons/register";
import "./styles/main.css";

registerHugeIcons();
createApp(App).mount("#app");
