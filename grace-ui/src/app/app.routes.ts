import { Routes } from '@angular/router';

export const routes: Routes = [
  {
    path: 'logs',
    loadComponent: () => import('./features/logs/logs').then((m) => m.Logs),
  },
  {
    path: 'devices',
    loadComponent: () => import('./features/devices/devices').then((m) => m.Devices),
  },
];
