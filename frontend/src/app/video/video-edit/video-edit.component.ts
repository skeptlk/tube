import { Component, OnInit } from '@angular/core';
import { VideoService } from 'src/app/services';
import { Video } from '../../models';


@Component({
    selector: 'app-video-edit',
    templateUrl: './video-edit.component.html',
    styleUrls: ['./video-edit.component.scss']
})
export class VideoEditComponent implements OnInit {

    constructor(private videoService: VideoService) { }

    ngOnInit(): void {
    }

}
