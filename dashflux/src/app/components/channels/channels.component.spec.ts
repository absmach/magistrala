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

import { MaterialModule } from '../../core/material/material.module';
import { AuthenticationService } from '../../core/services/auth/authentication.service';
import { TokenStorage } from '../../core/services/auth/token-storage.service';
import { ChannelsService } from '../../core/services/channels/channels.service';
import { ThingsService } from '../../core/services/things/things.service';
import { ChannelsComponent } from './channels.component';
import { ThingsStore } from '../../core/store/things.store';
import { UiStore } from '../../core/store/ui.store';
import { ChannelsStore } from '../../core/store/channels.store';

describe('ChannelsComponent', () => {
  let component: ChannelsComponent;
  let fixture: ComponentFixture<ChannelsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ ChannelsComponent ],
      imports: [
        MaterialModule,
        MatDialogModule,
        HttpClientModule,
        RouterTestingModule,
        FormsModule,
        ReactiveFormsModule,
        NoopAnimationsModule,
      ],
      providers: [
        {
          provide: ThingsStore,
          useClass: class {
            getThings = jasmine.createSpy('getThings');
          }
        },
        UiStore,
        {
          provide: ChannelsStore,
          useClass: class {
            getChannels = jasmine.createSpy('getChannels');
          }
        },
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
    fixture = TestBed.createComponent(ChannelsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});
