import { Component, model } from '@angular/core';
import { RouterModule } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { DialogModule } from 'primeng/dialog';
import { MenubarModule } from 'primeng/menubar';
import { SplitButtonModule } from 'primeng/splitbutton';

@Component({
  selector: 'grc-layout',
  templateUrl: './layout.html',
  imports: [MenubarModule, RouterModule, ButtonModule, DialogModule, SplitButtonModule],
})

export class Layout {
  addSystemDialogVisible = model(false);
  items = [
    {
      label: 'Home',
      icon: 'pi pi-home',
      routerLink: '/',
    },
    {
      label: 'Devices',
      icon: 'pi pi-desktop',
      routerLink: '/devices',
    },
    {
      label: 'Logs',
      icon: 'pi pi-list',
      routerLink: '/logs',
    },
  ];

  systems = [
    {
      label: 'Windows',
      icon: 'pi pi-windows',  
      command: () => this.getWindowsCommand(),
    },
  ];

  addSystem() {
    this.addSystemDialogVisible.set(true);
  }

  getLinuxCommand() {
    navigator.clipboard.writeText("linux command");
  }

  getWindowsCommand() {
    navigator.clipboard.writeText("windows command");
  }
} 