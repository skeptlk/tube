import { Component, OnInit } from '@angular/core';
import { ActivatedRoute } from '@angular/router';
import { VideoService } from '../services';
import { Video } from '../models';

@Component({
    selector: 'video-page',
    templateUrl: './video.component.html',
    styleUrls: ['./video.component.scss']
})
export class VideoComponent implements OnInit {

    video: Video = new Video({});
    id: number;

    constructor(
        public videoService: VideoService, 
        private route: ActivatedRoute) { }

    ngOnInit(): void {
        this.id = +this.route.snapshot.paramMap.get('id');
        this.videoService
            .getInfo(this.id)
            .subscribe(vid => this.video = vid);
    }

}
