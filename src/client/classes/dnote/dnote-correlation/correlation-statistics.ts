import type { DnoteCorrelationCell, DnoteCorrelationMethod } from "../dnote-correlation"

/**
 * 相関係数とその検定量を求める純粋な統計関数群。
 *
 * gkill はサーバーAPIを持たずクライアントで集計するため、統計ライブラリを足さずに
 * 必要な3つ（相関係数・p値・信頼区間）だけを自前で持つ。Kyou にもVueにも依存しない。
 */

/** 標準正規分布の両側95%点。Fisher z変換した相関係数の信頼区間に使う */
const fisher_95_percent_z = 1.959963984540054

/**
 * log(Γ(x))。ベータ関数の正規化に使う。
 *
 * Lanczos近似（g=7, n=9）。係数は近似式に固有の定数で、意味のある単位を持たない。
 * Γ(x) をそのまま計算すると標本数が増えたとたんに倍精度の範囲を超えるので、
 * 常に対数のまま扱って最後に exp する。
 * x<0.5 では反射公式 Γ(x)Γ(1-x)=π/sin(πx) で右半平面に折り返す（左半平面は近似の精度が出ない）。
 */
function log_gamma(value: number): number {
    const coefficients = [
        676.5203681218851,
        -1259.1392167224028,
        771.32342877765313,
        -176.61502916214059,
        12.507343278686905,
        -0.13857109526572012,
        9.984369578019572e-6,
        1.5056327351493116e-7,
    ]
    if (value < 0.5) {
        return Math.log(Math.PI) - Math.log(Math.sin(Math.PI * value)) - log_gamma(1 - value)
    }
    let x = 0.9999999999998099
    const shifted = value - 1
    for (let i = 0; i < coefficients.length; i++) {
        x += coefficients[i] / (shifted + i + 1)
    }
    const t = shifted + coefficients.length - 0.5
    return 0.5 * Math.log(2 * Math.PI) + (shifted + 0.5) * Math.log(t) - t + Math.log(x)
}

/**
 * 正則化不完全ベータ関数の連分数部分（Lentz法）。
 *
 * epsilon は倍精度の刻み幅に対する打ち切り条件で、これ以上細かくしても値が動かない。
 * smallest は「0で割らないための下駄」。Lentz法は途中の分母が0に落ちると発散するので、
 * 0になった項を極小値へ丸めて計算を続ける（この置き換えは最終値に影響しない）。
 * 200回は倍精度で収束しない入力が実質存在しない上限。
 */
function beta_continued_fraction(a: number, b: number, x: number): number {
    const max_iterations = 200
    const epsilon = 3e-14
    const smallest = 1e-300
    const qab = a + b
    const qap = a + 1
    const qam = a - 1
    let c = 1
    let d = 1 - qab * x / qap
    if (Math.abs(d) < smallest) d = smallest
    d = 1 / d
    let h = d

    for (let m = 1; m <= max_iterations; m++) {
        const m2 = 2 * m
        let aa = m * (b - m) * x / ((qam + m2) * (a + m2))
        d = 1 + aa * d
        if (Math.abs(d) < smallest) d = smallest
        c = 1 + aa / c
        if (Math.abs(c) < smallest) c = smallest
        d = 1 / d
        h *= d * c

        aa = -(a + m) * (qab + m) * x / ((a + m2) * (qap + m2))
        d = 1 + aa * d
        if (Math.abs(d) < smallest) d = smallest
        c = 1 + aa / c
        if (Math.abs(c) < smallest) c = smallest
        d = 1 / d
        const delta = d * c
        h *= delta
        if (Math.abs(delta - 1) < epsilon) break
    }
    return h
}

/**
 * 正則化不完全ベータ関数 I_x(a, b)。t分布の裾確率＝p値の実体。
 *
 * 連分数は x が小さい側でしか速く収束しないので、x が大きいときは
 * I_x(a,b) = 1 - I_(1-x)(b,a) の対称性で小さい側へ移してから計算する。
 */
function regularized_incomplete_beta(x: number, a: number, b: number): number {
    if (x <= 0) return 0
    if (x >= 1) return 1
    const front = Math.exp(log_gamma(a + b) - log_gamma(a) - log_gamma(b) + a * Math.log(x) + b * Math.log(1 - x))
    if (x < (a + 1) / (a + b + 2)) {
        return front * beta_continued_fraction(a, b, x) / a
    }
    return 1 - front * beta_continued_fraction(b, a, 1 - x) / b
}

/**
 * 昇順の順位を返す。Spearman相関で値そのものの代わりに使う。
 * 同値は平均順位を割り当てる（順位を先着順にすると、同じ値の並び順で相関が変わってしまう）。
 */
function rank(values: Array<number>): Array<number> {
    const indexed = values.map((value, index) => ({ value, index })).sort((a, b) => a.value - b.value)
    const ranks = new Array<number>(values.length)
    let start = 0
    while (start < indexed.length) {
        let end = start + 1
        while (end < indexed.length && indexed[end].value === indexed[start].value) end++
        const average_rank = (start + 1 + end) / 2
        for (let i = start; i < end; i++) ranks[indexed[i].index] = average_rank
        start = end
    }
    return ranks
}

/**
 * Pearson積率相関係数。算出できないときは null を返す。
 *
 * 3点未満は相関を語れないので null。片方が定数だと分母が0になるのでこれも null
 * （「相関が0」ではなく「求まらない」なので、0とは区別してヒートマップの色を分ける）。
 * 丸め誤差で ±1 をわずかに超えることがあるため、最後に範囲へ収める。
 */
export function pearson_correlation(xs: Array<number>, ys: Array<number>): number | null {
    if (xs.length !== ys.length || xs.length < 3) return null
    const x_mean = xs.reduce((sum, value) => sum + value, 0) / xs.length
    const y_mean = ys.reduce((sum, value) => sum + value, 0) / ys.length
    let numerator = 0
    let x_square_sum = 0
    let y_square_sum = 0
    for (let i = 0; i < xs.length; i++) {
        const x_delta = xs[i] - x_mean
        const y_delta = ys[i] - y_mean
        numerator += x_delta * y_delta
        x_square_sum += x_delta * x_delta
        y_square_sum += y_delta * y_delta
    }
    if (x_square_sum === 0 || y_square_sum === 0) return null
    const coefficient = numerator / Math.sqrt(x_square_sum * y_square_sum)
    return Number.isFinite(coefficient) ? Math.max(-1, Math.min(1, coefficient)) : null
}

/**
 * 相関係数・p値・95%信頼区間をまとめて求める。
 *
 * Spearman は「順位に対する Pearson」なので、順位へ変換して同じ計算に載せる。
 * p値は t = r√(df/(1-r²)) の両側確率を正則化不完全ベータで直接求める。
 * |r|=1 のときは t が発散するので p=0 を確定値として返す。
 * 信頼区間は Fisher z 変換（atanh）した空間で ±z/√(n-3) して戻す。
 * n<4 だと √(n-3) が0以下になり区間が定義できないため null（区間なし）にする。
 */
export function correlation_statistics(xs: Array<number>, ys: Array<number>, method: DnoteCorrelationMethod): Pick<DnoteCorrelationCell, "coefficient" | "p_value" | "confidence_low" | "confidence_high" | "sample_size"> {
    const sample_size = xs.length
    const coefficient = method === "spearman"
        ? pearson_correlation(rank(xs), rank(ys))
        : pearson_correlation(xs, ys)
    if (coefficient === null) {
        return { coefficient: null, p_value: null, confidence_low: null, confidence_high: null, sample_size }
    }

    let p_value = 0
    if (Math.abs(coefficient) < 1) {
        const degrees_of_freedom = sample_size - 2
        const t_squared = coefficient * coefficient * degrees_of_freedom / (1 - coefficient * coefficient)
        p_value = Math.max(0, Math.min(1, regularized_incomplete_beta(degrees_of_freedom / (degrees_of_freedom + t_squared), degrees_of_freedom / 2, 0.5)))
    }

    let confidence_low: number | null = null
    let confidence_high: number | null = null
    if (sample_size >= 4) {
        if (Math.abs(coefficient) === 1) {
            confidence_low = coefficient
            confidence_high = coefficient
        } else {
            const fisher = Math.atanh(coefficient)
            const margin = fisher_95_percent_z / Math.sqrt(sample_size - 3)
            confidence_low = Math.tanh(fisher - margin)
            confidence_high = Math.tanh(fisher + margin)
        }
    }
    return { coefficient, p_value, confidence_low, confidence_high, sample_size }
}
