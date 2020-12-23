import { Component, ElementRef, OnInit, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { AuthService, VideoService } from '../services';
import { Video } from '../models';

@Component({
    selector: 'video-page',
    templateUrl: './video.component.html',
    styleUrls: ['./video.component.scss']
})
export class VideoComponent implements OnInit {

    video: Video;
    id: number;
    showOwnerControls: boolean;

    constructor(
        public videoService: VideoService,
        public auth: AuthService,
        private route: ActivatedRoute
    ) { }

    ngOnInit(): void {
        this.id = +this.route.snapshot.paramMap.get('id');
        this.videoService
            .getInfo(this.id)
            .subscribe(vid => {
                this.video = vid;
                this.showOwnerControls = (vid.userId === this.auth.currentUserValue.id);
            });
    }

}
