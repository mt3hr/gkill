#!/usr/bin/env node
// マニュアルの ASCII 代替表記（アクセント欠落）を是正する（fr / es）。
//
// 安全策: <pre>…</pre> / <code>…</code> / HTML タグ（href 等の属性を含む）を
//         プレースホルダで退避し、散文テキストにのみ単語境界で置換を適用する。
//         → コード例・URL・ファイル名・識別子は絶対に書き換わらない。
//
// 辞書は「実際に確認できた高頻度・曖昧さのない語」のみを収録した初回パス。
// 追加の是正は辞書を拡張して再実行する（要ネイティブレビュー）。
//
//   node src/tools/manual_ascii_fix.mjs         フラグメントを是正
//   node src/tools/manual_ascii_fix.mjs --dry   置換件数のみ表示（書き込みなし）

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..', '..')
const SRC_DIR = path.join(ROOT, 'resources', 'manual_src')

// wrong -> correct（大文字小文字は別エントリ。複数形・活用も明示）
const DICT = {
  fr: {
    donnees: 'données', donnee: 'donnée',
    taches: 'tâches', tache: 'tâche',
    parametres: 'paramètres', parametre: 'paramètre',
    modeles: 'modèles', modele: 'modèle',
    agregation: 'agrégation',
    repertoire: 'répertoire', repertoires: 'répertoires',
    integration: 'intégration', integrations: 'intégrations',
    associes: 'associés', associe: 'associé',
    recommande: 'recommandé', recommandee: 'recommandée',
    regulierement: 'régulièrement',
    particulierement: 'particulièrement',
    Gerer: 'Gérer', gerer: 'gérer',
    Maitriser: 'Maîtriser', maitriser: 'maîtriser',
    numero: 'numéro', apres: 'après',
    creer: 'créer', elements: 'éléments', element: 'élément',
    etapes: 'étapes', etape: 'étape',
    selectionner: 'sélectionner',
    precedent: 'précédent', precedente: 'précédente',
    reglages: 'réglages', reglage: 'réglage',
  },
  es: {
    configuracion: 'configuración', Configuracion: 'Configuración',
    agregacion: 'agregación', Agregacion: 'Agregación',
    integracion: 'integración', Integracion: 'Integración',
    gestion: 'gestión', Gestion: 'Gestión',
    periodicamente: 'periódicamente',
    numero: 'número', informacion: 'información', Informacion: 'Información',
    aplicacion: 'aplicación', Aplicacion: 'Aplicación',
    sesion: 'sesión', Sesion: 'Sesión',
    boton: 'botón', Boton: 'Botón',
    tambien: 'también', categoria: 'categoría',
    seleccion: 'selección', Seleccion: 'Selección',
    edicion: 'edición', Edicion: 'Edición',
    version: 'versión', Version: 'Versión',
  },
}

// 慣用句（単語境界では拾えないもの）
const PHRASES = {
  fr: [['a la fois', 'à la fois']],
  es: [],
}

const escapeRe = (s) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

// 退避に使うセンチネル（散文・数字と衝突しない一意トークン）
const SENT = (i) => '@@PROTECT' + i + 'END@@'

function protectRegions(text, store) {
  const patterns = [/<pre[\s\S]*?<\/pre>/g, /<code>[\s\S]*?<\/code>/g, /<[^>]+>/g]
  let t = text
  for (const re of patterns) {
    t = t.replace(re, (m) => {
      const key = SENT(store.length)
      store.push(m)
      return key
    })
  }
  return t
}

function restoreRegions(text, store) {
  let t = text
  for (let i = store.length - 1; i >= 0; i--) {
    t = t.split(SENT(i)).join(store[i])
  }
  return t
}

function fixLang(lang, dry) {
  const dict = DICT[lang]
  const phrases = PHRASES[lang] || []
  const langDir = path.join(SRC_DIR, lang)
  if (!fs.existsSync(langDir)) return 0
  let total = 0
  for (const page of fs.readdirSync(langDir).filter((f) => f.endsWith('.html'))) {
    const p = path.join(langDir, page)
    const raw = fs.readFileSync(p, 'utf8').replace(/\r\n/g, '\n')
    const store = []
    let prose = protectRegions(raw, store)
    let count = 0
    for (const [wrong, right] of Object.entries(dict)) {
      if (wrong === right) continue
      const re = new RegExp('\\b' + escapeRe(wrong) + '\\b', 'g')
      prose = prose.replace(re, () => { count++; return right })
    }
    for (const [wrong, right] of phrases) {
      const re = new RegExp(escapeRe(wrong), 'g')
      prose = prose.replace(re, () => { count++; return right })
    }
    const out = restoreRegions(prose, store)
    if (count > 0) {
      total += count
      if (!dry) fs.writeFileSync(p, out, 'utf8')
      console.log('  ' + lang + '/' + page + ': ' + count + '件')
    }
  }
  return total
}

function main() {
  const dry = process.argv.includes('--dry')
  console.log(dry ? 'ASCII是正（ドライラン）:' : 'ASCII是正:')
  let total = 0
  for (const lang of ['fr', 'es']) total += fixLang(lang, dry)
  console.log((dry ? '（ドライラン）' : '') + '合計 ' + total + ' 件' + (dry ? '（書き込みなし）' : ' 置換'))
}

main()
