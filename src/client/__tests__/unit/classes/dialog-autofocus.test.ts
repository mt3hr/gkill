/**
 * ダイアログを開いたときのフォーカス先選び (classes/dialog-autofocus.ts) の検証。
 *
 * 「本文の最初の入力欄」を素直に取ると事故る箇所が実際にいくつもある:
 * ヘッダの透過トグル (v-checkbox) が DOM 順で本文より前にあり、
 * 日付欄は readonly の見せかけ入力 (フォーカスするとピッカーが開く)、
 * 検索条件のキーワード欄は v-show の内側、v-select の input は打ち込めない。
 * 実行時に「変なところにカーソルが載る」形でしか気づけないのでここで固定する。
 */
import { afterEach, describe, expect, it } from 'vitest'

import { find_autofocus_target, has_focus_inside } from '@/classes/dialog-autofocus'

/** ダイアログのDOM。ヘッダ(透過チェックボックス+×)と本文、という実物と同じ構造にする */
function mount_dialog(body_html: string): HTMLElement {
    const root = document.createElement('div')
    root.className = 'gkill-floating-dialog'
    root.innerHTML = `
      <div class="gkill-floating-dialog__header">
        <div class="v-selection-control"><input type="checkbox" id="transparent_toggle"></div>
        <button type="button">close</button>
      </div>
      <div class="gkill-floating-dialog__body">${body_html}</div>
    `
    document.body.appendChild(root)
    return root
}

afterEach(() => {
    document.body.innerHTML = ''
})

describe('find_autofocus_target', () => {
    it('本文の最初のテキスト入力欄を返す', () => {
        const root = mount_dialog('<input type="text" id="first"><input type="text" id="second">')
        expect(find_autofocus_target(root)?.id).toBe('first')
    })

    it('ヘッダの透過チェックボックスを選ばない', () => {
        const root = mount_dialog('<input type="text" id="body_input">')
        expect(find_autofocus_target(root)?.id).toBe('body_input')
    })

    it('本文が無ければ null', () => {
        const root = document.createElement('div')
        root.innerHTML = '<input type="text" id="stray">'
        document.body.appendChild(root)
        expect(find_autofocus_target(root)).toBeNull()
    })

    it('root 自身が本文でもよい', () => {
        const body = document.createElement('div')
        body.className = 'gkill-floating-dialog__body'
        body.innerHTML = '<input type="text" id="direct">'
        document.body.appendChild(body)
        expect(find_autofocus_target(body)?.id).toBe('direct')
    })

    it('null を渡しても落ちない', () => {
        expect(find_autofocus_target(null)).toBeNull()
    })

    it('既に autofocus が付いていたら何もしない（Vuetify に任せる）', () => {
        const root = mount_dialog('<input type="text" id="plain"><textarea autofocus id="explicit"></textarea>')
        expect(find_autofocus_target(root)).toBeNull()
    })

    it('textarea も対象', () => {
        const root = mount_dialog('<textarea id="memo"></textarea>')
        expect(find_autofocus_target(root)?.id).toBe('memo')
    })

    it.each([
        { name: 'readonly の日付欄（フォーカスするとピッカーが開く）', html: '<input type="text" readonly id="skip">' },
        { name: 'disabled', html: '<input type="text" disabled id="skip">' },
        { name: 'v-show で隠れた欄', html: '<div style="display: none;"><input type="text" id="skip"></div>' },
        { name: 'visibility: hidden', html: '<div style="visibility: hidden;"><input type="text" id="skip"></div>' },
        { name: 'hidden 属性', html: '<input type="text" hidden id="skip">' },
        { name: 'type=hidden', html: '<input type="hidden" id="skip">' },
        { name: 'type=file（アップロードの隠し input）', html: '<input type="file" id="skip">' },
        { name: 'v-checkbox の内部 input', html: '<div class="v-selection-control"><input type="checkbox" id="skip"></div>' },
        { name: 'v-select の input（inputmode="none" で打ち込めない）', html: '<input type="text" inputmode="none" id="skip">' },
        { name: 'ボタン', html: '<button type="button" id="skip">save</button>' },
    ])('$name は飛ばして次の欄を選ぶ', ({ html }) => {
        const root = mount_dialog(`${html}<input type="text" id="wanted">`)
        expect(find_autofocus_target(root)?.id).toBe('wanted')
    })

    it('候補が無ければ null（確認ダイアログなど）', () => {
        const root = mount_dialog('<p>削除しますか</p><button type="button">はい</button>')
        expect(find_autofocus_target(root)).toBeNull()
    })

    it('v-autocomplete / v-combobox の input は対象（打ち込める）', () => {
        const root = mount_dialog('<div class="v-combobox"><input type="text" id="combo"></div>')
        expect(find_autofocus_target(root)?.id).toBe('combo')
    })

    it.each(['number', 'url', 'email', 'search', 'password', 'tel', 'date', 'time'])(
        'type=%s も対象', (type) => {
            const root = mount_dialog(`<input type="${type}" id="typed">`)
            expect(find_autofocus_target(root)?.id).toBe('typed')
        })
})

describe('has_focus_inside', () => {
    it('ダイアログ内の入力欄にフォーカスがあれば true', () => {
        const root = mount_dialog('<input type="text" id="focused">')
        const input = root.querySelector<HTMLInputElement>('#focused')
        input?.focus()
        expect(has_focus_inside(root)).toBe(true)
    })

    it('どこにもフォーカスしていなければ false', () => {
        const root = mount_dialog('<input type="text" id="unfocused">')
        expect(has_focus_inside(root)).toBe(false)
    })

    it('ボタンにフォーカスしているだけなら false（入力欄は未確定）', () => {
        const root = mount_dialog('<button type="button" id="btn">save</button><input type="text" id="later">')
        root.querySelector<HTMLButtonElement>('#btn')?.focus()
        expect(has_focus_inside(root)).toBe(false)
    })

    it('null を渡しても落ちない', () => {
        expect(has_focus_inside(null)).toBe(false)
    })
})
