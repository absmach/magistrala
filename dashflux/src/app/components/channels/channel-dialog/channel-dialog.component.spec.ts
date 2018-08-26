/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */
 
import { HttpClientModule } from '@angular/common/http';
import { async, ComponentFixture, TestBed } from '@angular/core/testing';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { MAT_DIALOG_DATA, MatDialogModule, MatDialogRef } from '@angular/material';
import { NoopAnimationsModule } from '@angular/platform-browser/animations';
import { RouterTestingModule } from '@angular/router/testing';

import { MaterialModule } from '../../../core/material/material.module';
import { AuthenticationService } from '../../../core/services/auth/authentication.service';
import { TokenStorage } from '../../../core/services/auth/token-storage.service';
import { ChannelsService } from '../../../core/services/channels/channels.service';
import { ThingsService } from '../../../core/services/things/things.service';
import { ChannelDialogComponent } from './channel-dialog.component';
import { ThingsStore } from '../../../core/store/things.store';
import { UiStore } from '../../../core/store/ui.store';

describe('ChannelDialogComponent', () => {
  let component: ChannelDialogComponent;
  let fixture: ComponentFixture<ChannelDialogComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ChannelDialogComponent ],
      imports: [
        MaterialModule,
        MatDialogModule,
        HttpClientModule,
        RouterTestingModule,
        FormsModule,
        ReactiveFormsModule,
        NoopAnimationsModule
      ],
      providers: [
        ThingsStore,
        UiStore,
        AuthenticationService,
        TokenStorage,
        ThingsService,
        ChannelsService,
        { provide: MatDialogRef, useValue: {} },
        { provide: MAT_DIALOG_DATA, useValue: [] },
      ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(ChannelDialogComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
