/*
 * Copyright (c) 2018
 * Mainflux
 *
 * SPDX-License-Identifier: Apache-2.0
 */
 
import { Component, OnInit } from '@angular/core';
import { MatDialog } from '@angular/material';
import { Observable } from 'rxjs/Observable';

import { ChannelsStore } from '../../core/store/channels.store';
import { ThingsStore } from '../../core/store/things.store';
import { Channel } from '../../core/store/models';
import { ConfirmationDialogComponent } from '../shared/confirmation-dialog/confirmation-dialog.component';
import { ChannelDialogComponent } from './channel-dialog/channel-dialog.component';

@Component({
  selector: 'app-channels',
  templateUrl: './channels.component.html',
  styleUrls: ['./channels.component.scss'],
})
export class ChannelsComponent implements OnInit {
  channels: Observable<Channel[]>;
  displayedColumns = ['id', 'name', 'connected', 'actions'];

  constructor(
    private dialog: MatDialog,
    public thingsStore: ThingsStore,
    public channelsStore: ChannelsStore,
  ) { }

  ngOnInit() {
    this.channelsStore.getChannels();
    this.thingsStore.getThings();
  }

  addChannel() {
    const dialogRef = this.dialog.open(ChannelDialogComponent);

    dialogRef.componentInstance.submit.subscribe((channel: Channel) => {
      this.channelsStore.addChannel(channel);
    });
  }

  editChannel(channel: Channel) {
    const dialogRef = this.dialog.open(ChannelDialogComponent, {
      data: channel
    });

    dialogRef.componentInstance.submit.subscribe((editedChannel: Channel) => {
      this.channelsStore.editChannel(editedChannel);
    });
  }

  deleteChannel(channel: Channel) {
    const dialogRef = this.dialog.open(ConfirmationDialogComponent, {
      data: {
        question: 'Are you sure you want to delete the channel?'
      }
    });

    dialogRef.afterClosed().subscribe((result) => {
      if (result) {
        this.channelsStore.deleteChannel(channel);
      }
    });
  }
}
