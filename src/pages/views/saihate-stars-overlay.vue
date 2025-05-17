<template>
  <div class="camera-wrap">
    <!-- <div class="moon"></div> -->
    <div ref="starField" class="star-field"></div>
  </div>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import { onMounted, ref } from 'vue'

const starField = ref<HTMLElement | null>(null)

function createStar(className: string, top: number, left: number, duration?: number) {
  const star = document.createElement('div')
  star.className = className
  star.style.top = `${top}px`
  star.style.left = `${left}px`
  if (duration) star.style.animationDuration = `${duration}s`
  starField.value?.appendChild(star)
}

function createShootingStar() {
  const star = document.createElement('div')
  star.className = 'shooting-star'
  const length = Math.random() * 100 + 100
  const startX = Math.random() * window.innerWidth
  const startY = Math.random() * window.innerHeight * 0.5
  const duration = (Math.random() * 0.5 + 0.5).toFixed(2)

  star.style.width = `${length}px`
  star.style.height = '2px'
  star.style.position = 'absolute'
  star.style.top = `${startY}px`
  star.style.left = `${startX}px`
  star.style.transform = 'rotate(135deg)'
  star.style.background = 'linear-gradient(135deg, rgba(255,255,255,0) 0%, rgba(255,255,255,0.6) 50%, white 100%)'
  star.style.animation = `shooting ${duration}s ease-out forwards`
  star.style.pointerEvents = 'none'
  star.style.opacity = '0'
  starField.value?.appendChild(star)

  setTimeout(() => star.remove(), +duration * 1000)
}

function loopShootingStars() {
  const count = Math.floor(Math.random() * 3) + 1
  for (let i = 0; i < count; i++) {
    setTimeout(createShootingStar, Math.random() * 300)
  }
  setTimeout(loopShootingStars, Math.random() * 1500 + 500)
}

onMounted(() => {
  const h = window.innerHeight
  const w = window.innerWidth

  for (let i = 0; i < 100; i++) {
    createStar('background-star', Math.random() * h, Math.random() * w, Math.random() * 2 + 1)
  }
  for (let i = 0; i < 5; i++) {
    createStar('background-star red-star', Math.random() * h, Math.random() * w)
    createStar('background-star big-star', Math.random() * h, Math.random() * w)
    createStar('background-star blue-star', Math.random() * h, Math.random() * w)
  }

  loopShootingStars()
})
</script>

<style>
.camera-wrap {
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  pointer-events: none;
  animation: cameraShake 10s infinite ease-in-out;
  overflow: hidden;
  /* z-index: 1; */
}

@keyframes cameraShake {
  0% { transform: translate(0px, 0px); }
  25% { transform: translate(1px, -1px); }
  50% { transform: translate(-1px, 1px); }
  75% { transform: translate(1px, 0px); }
  100% { transform: translate(0px, 0px); }
}

.star-field {
  position: absolute;
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
}

.background-star {
  position: absolute;
  width: 2px;
  height: 2px;
  background: white;
  border-radius: 50%;
  animation: twinkle 2s infinite;
  opacity: 0.8;
}

@keyframes twinkle {
  0%, 100% { opacity: 0.3; }
  50% { opacity: 1; }
}

.red-star {
  background: red;
  width: 3px;
  height: 3px;
  animation: twinkleRed 3s infinite;
}

@keyframes twinkleRed {
  0%, 100% { opacity: 0.1; }
  50% { opacity: 0.9; }
}

.big-star {
  width: 4px;
  height: 4px;
  animation: twinkleBig 4s infinite;
}

@keyframes twinkleBig {
  0%, 100% { opacity: 0.5; }
  50% { opacity: 1; }
}

.blue-star {
  background: #66ccff;
  width: 3px;
  height: 3px;
  animation: twinkleBlue 3.5s infinite;
}

@keyframes twinkleBlue {
  0%, 100% { opacity: 0.2; }
  50% { opacity: 0.8; }
}

.moon {
  position: absolute;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: radial-gradient(circle, #fffbe6 0%, #d0cfcf 100%);
  top: 50px;
  left: 50px;
  box-shadow: 0 0 20px rgba(255, 255, 200, 0.5);
  z-index: 2;
}

.shooting-star {
  animation: shooting 1s ease-out forwards;
}

@keyframes shooting {
  0% {
    opacity: 1;
    transform: translate(0, 0) rotate(135deg);
  }
  100% {
    opacity: 0;
    transform: translate(-500px, 500px) rotate(135deg);
  }
}
</style>
