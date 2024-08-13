import { createRouter, createWebHistory } from 'vue-router'

import login_page from '../pages/login-page.vue'
import kftl_page from '../pages/kftl-page.vue'
import mi_page from '../pages/mi-page.vue'
import rykv_page from '../pages/rykv-page.vue'
import saihate_page from '../pages/saihate-page.vue'
import set_new_password_page from '../pages/set-new-password-page.vue'
import shared_mi_page from '../pages/shared-mi-page.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'login',
      component: login_page,
    },
    {
      path: '/kftl',
      name: 'kftl',
      component: kftl_page,
    },
    {
      path: '/mi',
      name: 'mi',
      component: mi_page,
    },
    {
      path: '/rykv',
      name: 'rykv',
      component: rykv_page,
    },
    {
      path: '/saihate',
      name: 'saihate',
      component: saihate_page,
    },
    {
      path: '/',
      name: 'set_new_password',
      component: set_new_password_page,
    },
    {
      path: '/shared_mi',
      name: 'shared_mi',
      component: shared_mi_page,
    },
  ]
})

export default router
