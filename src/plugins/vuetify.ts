import "vuetify/styles"
import { createVuetify, type ThemeDefinition } from "vuetify"
import * as components from "vuetify/components"
import * as directives from "vuetify/directives"
import { aliases, mdi } from "vuetify/iconsets/mdi"

const gkill_theme: ThemeDefinition = {
	dark: false,
	colors: {
		background: '#ffffff',
		surface: '#ffffff',
		primary: '#2672ed',
		'primary-darken-1': '#2672ed',
		secondary: '#26c2ed',
		'secondary-darken-1': '#26c2ed',
		error: '#B00020',
		info: '#2196F3',
		success: '#4CAF50',
		warning: '#FB8C00',
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
			gkill_theme
		},
	},
})

export default vuetify

