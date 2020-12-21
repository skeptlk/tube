import { Component, OnInit } from '@angular/core';
import { Video } from '../models';
import { VideoService } from '../services';

@Component({
    selector: 'app-home',
    templateUrl: './home.component.html',
    styleUrls: ['./home.component.scss']
})
export class HomeComponent implements OnInit {

    videos: Video[] = [];

    constructor(public vidService: VideoService) { }

    ngOnInit(): void {
        this.vidService.list().subscribe((vids) => {
            this.videos = vids
        });
    }

}
