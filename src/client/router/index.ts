import { createRouter, createWebHistory } from 'vue-router'

import login_page from '../pages/login-page.vue'
import kftl_page from '../pages/kftl-page.vue'
import mi_page from '../pages/mi-page.vue'
import rykv_page from '../pages/rykv-page.vue'
import kyou_page from '../pages/kyou-page.vue'
import saihate_page from '../pages/saihate-page.vue'
import set_new_password_page from '../pages/set-new-password-page.vue'
import shared_page from '../pages/shared-page.vue'
import plaing_timeis_page from '@/pages/plaing-time-is-page.vue'
import mkfl_page from '@/pages/mkfl-page.vue'
import register_first_account_page from '@/pages/register-first-account-page.vue'
import dashboard_page from '@/pages/dashboard-page.vue'
import rudbeckia_page from '@/pages/rudbeckia-page.vue'

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
      path: '/kyou',
      name: 'kyou',
      component: kyou_page,
    },
    {
      path: '/mkfl',
      name: 'mkfl',
      component: mkfl_page,
    },
    {
      path: '/plaing',
      name: 'plaing',
      component: plaing_timeis_page,
    },
    {
      path: '/saihate',
      name: 'saihate',
      component: saihate_page,
    },
    {
      path: '/dashboard',
      name: 'dashboard',
      component: dashboard_page,
    },
    {
      // 開発コードは rudbeckia。ユーザ向けの呼び名は「ポート」
      path: '/rudbeckia',
      name: 'rudbeckia',
      component: rudbeckia_page,
    },
    {
      path: '/set_new_password',
      name: 'set_new_password',
      component: set_new_password_page,
    },
    {
      path: '/register_first_account',
      name: 'register_first_account',
      component: register_first_account_page
    },
    {
      // 旧パス。ブックマークや古い資料からの流入を新パスへ寄せる。
      // reset_token クエリを落とすと初回セットアップが通らないので引き継ぐこと。
      path: '/regist_first_account',
      redirect: (to) => ({ path: '/register_first_account', query: to.query }),
    },
    {
      path: '/shared_page',
      name: 'shared_page',
      component: shared_page,
    },
    {
      // 旧パス。共有URLは配布済みで再発行できないので受け続ける。
      // **コンポーネントの setup から router.replace してはいけない。**
      // ここは `<script setup>` に top-level await のある非同期コンポーネントで、
      // 初回ナビゲーションの解決中にその中から新しいナビゲーションを始めると
      // 遷移が完了しなくなる（page.goto が60秒待っても返らない）。
      // これまでは share_id が無いと `query.share_id!.toString()` が throw して
      // setup ごと落ちていたため、**redirect 自体が一度も走っておらず**露見しなかった。
      // /regist_first_account と同じく、ルータの redirect で置き換える。
      // query をそのまま引き継ぐので share_id も残る
      path: '/shared_mi',
      redirect: (to) => ({ path: '/shared_page', query: to.query }),
    },
  ]
})

export default router
