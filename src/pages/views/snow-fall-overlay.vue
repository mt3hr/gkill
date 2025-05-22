<template>
    <div class="snow-container">
        <div ref="snowField" class="snow-field"></div>
    </div>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import { onMounted, ref } from 'vue'

const snowField = ref<HTMLElement | null>(null)

function createSnowflake() {
    const flake = document.createElement('div')
    flake.className = 'snowflake'

    const size = Math.random() * 6 + 2
    const left = Math.random() * window.innerWidth
    const duration = Math.random() * 5 + 5

    flake.style.width = `${size}px`
    flake.style.height = `${size}px`
    flake.style.left = `${left}px`
    flake.style.animationDuration = `${duration}s`

    snowField.value?.appendChild(flake)

    setTimeout(() => flake.remove(), duration * 1000)
}

function loopSnowfall() {
    createSnowflake()
    setTimeout(loopSnowfall, 100)
}

onMounted(() => {
    loopSnowfall()
})
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
