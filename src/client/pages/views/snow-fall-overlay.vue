<template>
    <div class="snow-container">
        <div ref="snowField" class="snow-field"></div>
    </div>
</template>

<script setup lang="ts">
import { useSnowFallOverlay } from '@/classes/use-snow-fall-overlay'

const { snowField } = useSnowFallOverlay()
</script>

<style>
.snow-container {
    position: fixed;
    top: 0;
    left: 0;
    width: 100vw;
    height: 100vh;
    pointer-events: none;
    overflow: hidden;
    z-index: -100000000;
    background-color: white;
}

.snow-field {
    position: absolute;
    width: 100%;
    height: 100%;
    top: 0;
    left: 0;
}

.snowflake {
    position: absolute;
    top: -10px;
    background: rgba(200, 200, 255, 0.8);
    border-radius: 50%;
    animation: fall linear forwards;
}

@keyframes fall {
    to {
        transform: translateY(100vh);
        opacity: 0.3;
    }
}
</style>
