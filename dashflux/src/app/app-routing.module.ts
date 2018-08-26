/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */

import { CommonModule } from '@angular/common';
import { NgModule } from '@angular/core';
import { Route, RouterModule } from '@angular/router';
import { ProtectedGuard, PublicGuard } from 'ngx-auth';

import { LoginComponent } from './components/auth/login/login.component';
import { SignupComponent } from './components/auth/signup/signup.component';
import { ChannelsComponent } from './components/channels/channels.component';
import { ThingsComponent } from './components/things/things.component';

const routes: Route[] = [
  { path: '', redirectTo: 'things', pathMatch: 'full'},
  { path: 'login', component: LoginComponent, canActivate: [PublicGuard]},
  { path: 'signup', component: SignupComponent, canActivate: [PublicGuard]},
  { path: 'things', component: ThingsComponent, canActivate: [ProtectedGuard]},
  { path: 'channels', component: ChannelsComponent, canActivate: [ProtectedGuard]}
];

@NgModule({
  imports: [
    CommonModule,
    RouterModule.forRoot(routes)
  ],
  exports: [
    RouterModule
  ]
})
export class AppRoutingModule { }
