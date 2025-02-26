import "vuetify/styles"
import { createVuetify, type ThemeDefinition } from "vuetify"
import * as components from "vuetify/components"
import * as directives from "vuetify/directives"
import { aliases, mdi } from "vuetify/iconsets/mdi"

const gkill_theme: ThemeDefinition = {
	dark: false,
	colors: {
		background: '#ffffff',
		'background-focused': '#C0C0C0',
		surface: '#ffffff',
		primary: '#2672ed',
		'primary-darken-1': '#2672ed',
		secondary: '#26c2ed',
		'secondary-darken-1': '#26c2ed',
		error: '#B00020',
		info: '#2196F3',
		success: '#4CAF50',
		warning: '#FB8C00',
		highlight: '#8cffbe',
	},
}

const gkill_dark_theme: ThemeDefinition = {
	dark: true,
	colors: {
		background: '#212121',
		'background-focused': '#4D4D4D',
		surface: '#212121',
		primary: '#2672ed',
		'primary-darken-1': '#2672ed',
		secondary: '#1d96b8',
		'secondary-darken-1': '#1d96b8',
		'attached-text-background': '#eee',
		error: '#7a0117',
		info: '#1765a3',
		success: '#218025',
		warning: '#9e5800',
		highlight: '#60ab80',
	},
}

const vuetify = createVuetify({
	components,
	directives,
	icons: {
		defaultSet: "mdi",
		aliases,
		sets: {
			mdi,
		},
	},
	theme: {
		defaultTheme: 'gkill_theme',
		themes: {
			gkill_theme,
			gkill_dark_theme,
		},
	},
})

export default vuetify

