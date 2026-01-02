import { provideHttpClient } from '@angular/common/http';
import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideRouter, withComponentInputBinding } from '@angular/router';
import { definePreset, palette } from '@primeuix/themes';
import Aura from '@primeuix/themes/aura';
import { providePrimeNG } from 'primeng/config';

import { routes } from './app.routes';

const GracePreset = definePreset(Aura, {
  primitive: {
    gold: palette('#ba9638'),
    amber: palette('#E8B923'),
    slate: palette('#1a1d23'),
    green: palette('#7CB342'),
    red: palette('#C75B5B'),
  },
  semantic: {
    primary: {
      50: '{gold.50}',
      100: '{gold.100}',
      200: '{gold.200}',
      300: '{gold.300}',
      400: '{gold.400}',
      500: '{gold.500}',
      600: '{gold.600}',
      700: '{gold.700}',
      800: '{gold.800}',
      900: '{gold.900}',
      950: '{gold.950}',
    },
  },
  components: {
    menubar: {
      item: {
        icon: {
          focusColor: '{gold.400}',
          color: '{gold.500}',
        },
      },
    },
  },
});

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideHttpClient(),
    provideRouter(routes, withComponentInputBinding()),
    providePrimeNG({
      theme: {
        preset: GracePreset,
        options: {
          darkModeSelector: '.dark',
        },
      },
    }),
  ],
};
