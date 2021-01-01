import { Component, Input, OnInit } from '@angular/core';
import { CommentsService } from '../services';
import { Comment, User } from '../models';

@Component({
    selector: 'video-comments',
    templateUrl: './comments.component.html',
    styleUrls: ['./comments.component.scss']
})
export class CommentsComponent implements OnInit {
    @Input() videoId: number;
    @Input() userId: number;

    newComment: Comment = new Comment();

    constructor(public commentsService: CommentsService) { }

    ngOnInit() {
        this.commentsService
            .list(this.videoId)
            .toPromise();
    }

    submitComment() {
        this.commentsService
            .create(this.newComment.text, this.videoId)
            .toPromise();

        this.newComment = new Comment();
    }

}
