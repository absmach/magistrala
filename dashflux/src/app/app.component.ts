/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */

import { Component, OnInit } from '@angular/core';
import { MatSnackBar } from '@angular/material';
import { reaction } from 'mobx';

import { UiStore } from './core/store/ui.store';
import { AuthStore } from './core/store/auth.store';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss']
})
export class AppComponent implements OnInit {
  constructor(
    private snackBar: MatSnackBar,
    public uiStore: UiStore,
    public authStore: AuthStore,
  ) { }

  ngOnInit() {
    reaction(() => this.authStore.authError, (authError) => {
      if (authError) {
        this.snackBar.open(authError, '', {
          duration: 3000
        });
      }
    });
  }

  logout() {
    this.authStore.logout();
  }
}
