/**
 * 子コンポーネントへのテンプレート ref の型。
 *
 * 添字シグネチャにしてあるので、子が公開しているメソッド/プロパティを
 * オプショナルチェーンで呼べる（例: `ref.value?.show(e)`）。
 * `InstanceType<typeof Child>` を使うと親子で import が循環するので、
 * それを避けるためにこの形にしている。
 *
 * **`src/client` の製品コードに残っている `any` はこの1つだけ。**
 * ほかは `unknown` + 絞り込み、または `as unknown as 具体型` へ寄せてある
 * （`@typescript-eslint/no-explicit-any` は error 指定で、抑制はここにしか無い）。
 * 新しく `any` を足したくなったら、まずここへ寄せられないかを考えること。
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type ComponentRef = Record<string, any>
